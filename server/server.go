package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	proto "grpc/grpc"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type AuctionServer struct {
	Bids              map[string]int32           // map[bidderID]bidAmount
	Responses         map[string]*proto.Response // map[requestID]*Response
	Lock              sync.Mutex
	AuctionEnd        time.Time
	isCurrentlyLeader bool
	proto.UnimplementedAuctionServer
}

func NewAuctionServer(leaseDuration time.Duration) *AuctionServer {
	return &AuctionServer{
		Bids:              make(map[string]int32),
		Responses:         make(map[string]*proto.Response),
		isCurrentlyLeader: true,
	}
}
func (s *AuctionServer) CheckHealth(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	// Here you can add any logic to determine the health of your server.
	// For example, you can check database connections, external dependencies, etc.
	// For simplicity, this example will just return a healthy status.

	return &proto.HealthCheckResponse{Healthy: true}, nil
}

func setupGracefulShutdown(mainServer *grpc.Server, auctionServer *AuctionServer) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		fmt.Println("Server is shutting down...")
		auctionServer.gracefulShutdown(mainServer)
	}()
}
func (s *AuctionServer) Bid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	// Check if the auction has ended
	if time.Now().After(s.AuctionEnd) {
		return &proto.BidResponse{Outcome: "Auction has ended. Bids are no longer accepted."}, nil
	}

	// Validate the bid amount
	currentHighestBid, exists := s.Bids[req.BidderId]
	if !exists || req.Amount > currentHighestBid {
		s.Bids[req.BidderId] = req.Amount
		return &proto.BidResponse{Outcome: "Bid accepted."}, nil
	}

	return &proto.BidResponse{Outcome: "Bid too low. Please place a higher bid."}, nil
}

func (s *AuctionServer) Result(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	// Check if the auction has ended
	if time.Now().After(s.AuctionEnd) {
		// Find the highest bid
		var winningBid int32
		var winner string
		for bidderId, bidAmount := range s.Bids {
			if bidAmount > winningBid {
				winningBid = bidAmount
				winner = bidderId
			}
		}
		return &proto.ResultResponse{Outcome: fmt.Sprintf("Auction ended. Winner: %s with bid: %d", winner, winningBid)}, nil
	}

	// Auction is still ongoing
	return &proto.ResultResponse{Outcome: "Auction is still ongoing. Result not available yet."}, nil
}

var replicaManagerAddresses = []string{"localhost:50040"}

func (s *AuctionServer) tryHeartbeat() bool {
	// Simulate checking if the main server is still alive
	// For simplicity, let's assume it always succeeds
	return true
}

func generateUniqueRequestID() string {
	return uuid.New().String()
}
func (s *AuctionServer) gracefulShutdown(mainServer *grpc.Server) {
	fmt.Println("Notifying replica manager and transferring leadership...")
	s.notifyReplicaManager("localhost:50040")

	fmt.Println("Performing additional shutdown tasks...")
	// Here you can add any additional cleanup logic needed before shutting down the server

	fmt.Println("Shutting down the gRPC server gracefully...")
	mainServer.GracefulStop()
}

func (s *AuctionServer) notifyReplicaManager(address string) {
	conn, err := grpc.Dial(address, grpc.WithInsecure()) // Adjust as necessary for security
	if err != nil {
		log.Printf("Failed to dial replica manager at %s: %v", address, err)
		return
	}
	defer conn.Close()

	client := proto.NewAuctionClient(conn)
	_, err = client.HandleLeadershipTransfer(context.Background(), &proto.LeadershipTransferRequest{})
	if err != nil {
		log.Printf("Failed to transfer leadership to replica manager at %s: %v", address, err)
	} else {
		fmt.Printf("Leadership successfully transferred to replica manager at %s\n", address)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	mainServer := grpc.NewServer()
	auctionServer := NewAuctionServer(10 * time.Second)

	proto.RegisterAuctionServer(mainServer, auctionServer)

	// Setup graceful shutdown
	setupGracefulShutdown(mainServer, auctionServer)

	// Set auction end time and start server
	auctionServer.AuctionEnd = time.Now().Add(100 * time.Second)
	log.Printf("Server created. Listening on port %s", lis.Addr().String())

	if err := mainServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
