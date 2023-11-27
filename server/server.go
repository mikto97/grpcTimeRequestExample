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
	LeaderLease       time.Time
	LeaseDuration     time.Duration
	proto.UnimplementedAuctionServer
}

func NewAuctionServer(leaseDuration time.Duration) *AuctionServer {
	return &AuctionServer{
		Bids:              make(map[string]int32),
		Responses:         make(map[string]*proto.Response),
		isCurrentlyLeader: false,
		LeaseDuration:     leaseDuration,
	}
}
func (s *AuctionServer) CheckHealth(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	// Here you can add any logic to determine the health of your server.
	// For example, you can check database connections, external dependencies, etc.
	// For simplicity, this example will just return a healthy status.

	return &proto.HealthCheckResponse{Healthy: true}, nil
}

func (s *AuctionServer) startLeaderElection() {
	// Start a goroutine for leader election
	go func() {
		// Start sending heartbeats to check main server's availability
		go s.sendHeartbeat()

		for {
			// Try to acquire the lease
			if s.tryAcquireLease() {
				s.isCurrentlyLeader = true
				fmt.Println("I am the leader!")
			} else {
				s.isCurrentlyLeader = false
			}

			// Wait for a short duration before attempting to acquire the lease again
			time.Sleep(s.LeaseDuration / 2)
		}
	}()
}

func (s *AuctionServer) IsLeader(ctx context.Context, req *proto.LeaderCheckRequest) (*proto.LeaderCheckResponse, error) {
	return &proto.LeaderCheckResponse{IsLeader: s.isCurrentlyLeader}, nil
}

// sendHeartbeat sends periodic heartbeats to check main server's availability
func (s *AuctionServer) sendHeartbeat() {
	for {
		// Implement logic to send a heartbeat to the main server
		// For simplicity, let's use a sleep to simulate heartbeat
		time.Sleep(s.LeaseDuration / 2)

		// If the main server is not responding, initiate leader election
		if !s.tryHeartbeat() {
			s.tryAcquireLease()
		}
	}
}

func (s *AuctionServer) tryAcquireLease() bool {
	// Simulate acquiring a lease by checking a shared resource (e.g., a file, database, or external service)
	// In a real-world scenario, you would use a distributed coordination service like etcd or ZooKeeper for lease acquisition

	// Check if the lease is currently held
	leaseHeld := s.checkLease()

	if !leaseHeld {
		// Lease is not held, try to acquire it
		if s.acquireLease() {
			// Successfully acquired the lease
			s.LeaderLease = time.Now().Add(s.LeaseDuration)
			return true
		} else {
			// Acquiring lease failed, initiate leader election
			s.startLeaderElection()
		}
	}

	return false
}

func (s *AuctionServer) checkLease() bool {
	// Simulate checking if the lease is currently held
	return time.Now().Before(s.LeaderLease)
}

// Implement acquireLease to acquire the lease
func (s *AuctionServer) acquireLease() bool {
	// Simulate acquiring the lease by setting the lease expiration time
	s.LeaderLease = time.Now().Add(s.LeaseDuration)
	return true
}

func setupGracefulShutdown(mainServer *grpc.Server, auctionServer *AuctionServer) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		fmt.Println("Server is shutting down...")

		// Perform necessary cleanup tasks
		// Notify replica managers about the shutdown
		auctionServer.gracefulShutdown()

		// Gracefully stop the gRPC server
		mainServer.GracefulStop()

		// Optionally, save current auction state to a file or database

		os.Exit(0)
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

func (s *AuctionServer) WaitForResponsesFromReplicaManagers(ctx context.Context, req *proto.Empty) (*proto.Responses, error) {
	// Acquire lock to prevent data race with other methods
	s.Lock.Lock()
	defer s.Lock.Unlock()

	fmt.Println("Waiting for responses in server.go from replica managers...")

	// Use channels and goroutines to simulate waiting for responses
	responseChannel := make(chan *proto.Response, len(replicaManagerAddresses))

	for _, replicaManagerAddress := range replicaManagerAddresses {
		go func(addr string) {
			// Implement logic to wait for responses from replica managers
			// This could involve waiting for a signal, checking a shared data structure, etc.
			// For simplicity, I'll just return a placeholder response for now.
			// Replace this with your actual implementation.
			response := &proto.Response{
				RequestId: generateUniqueRequestID(),
				Success:   true, // Set based on the success of replication
			}
			responseChannel <- response
		}(replicaManagerAddress)
	}

	// Collect responses from channels
	responses := make([]*proto.Response, 0, len(replicaManagerAddresses))
	for i := 0; i < len(replicaManagerAddresses); i++ {
		select {
		case response := <-responseChannel:
			// Check if the replica manager is still active
			if response != nil {
				s.Responses[response.RequestId] = response
				responses = append(responses, response)
			} else {
				// Handle the case where a replica manager has crashed
				// You may want to initiate recovery mechanisms or elect a new leader
				fmt.Println("Replica manager crashed!")
			}
		case <-time.After(2 * time.Second):
			// Handle the case where a replica manager does not respond within a timeout
			// You may want to initiate recovery mechanisms or elect a new leader
			fmt.Println("Replica manager timeout!")
		}
	}
	return &proto.Responses{Responses: responses}, nil

}

func (s *AuctionServer) tryHeartbeat() bool {
	// Simulate checking if the main server is still alive
	// For simplicity, let's assume it always succeeds
	return true
}

func generateUniqueRequestID() string {
	return uuid.New().String()
}
func (s *AuctionServer) gracefulShutdown() {
	// Notify replica managers about the shutdown and transfer leadership
	s.notifyReplicaManagers()

	// Additional shutdown logic...
}

func (s *AuctionServer) notifyReplicaManagers() {
	// Assuming a slice of addresses for the replica managers
	for _, address := range replicaManagerAddresses {
		conn, err := grpc.Dial(address, grpc.WithInsecure()) // Adjust as necessary for security
		if err != nil {
			log.Printf("Failed to dial replica manager at %s: %v", address, err)
			continue
		}
		defer conn.Close()

		client := proto.NewAuctionClient(conn)
		_, err = client.HandleLeadershipTransfer(context.Background(), &proto.LeadershipTransferRequest{})
		if err != nil {
			log.Printf("Failed to transfer leadership to replica manager at %s: %v", address, err)
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	mainServer := grpc.NewServer()
	auctionServer := NewAuctionServer(10 * time.Second)
	auctionServer.startLeaderElection()

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
