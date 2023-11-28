package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	proto "grpc/grpc"

	"google.golang.org/grpc"
)

type ReplicaManager struct {
	Responses         map[string]*proto.Response // map[requestID]*Response
	CurrentBids       map[string]int32           // map[bidderID]bidAmount
	isCurrentlyLeader bool                       // Simplified leadership logic
	AuctionEnd        time.Time
	Mutex             sync.Mutex // Mutex to protect shared resources
	proto.UnimplementedAuctionServer
}

// NewReplicaManager initializes a new ReplicaManager
func NewReplicaManager() *ReplicaManager {
	return &ReplicaManager{
		Responses:         make(map[string]*proto.Response),
		CurrentBids:       make(map[string]int32),
		isCurrentlyLeader: true, // Assuming it starts as leader
	}
}

func (r *ReplicaManager) ReplicateBidRequest(ctx context.Context, req *proto.BidRequest) (*proto.Response, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// Logic to handle replicated bid requests
	highestBid := r.CurrentBids[req.BidderId]
	message := "Bid accepted as the highest bid"
	success := true
	fmt.Println("Bid accepted as the highest bid")

	if req.Amount <= highestBid {
		message = "Bid too low. Current highest bid is: " + strconv.Itoa(int(highestBid))
		success = false
		fmt.Println("Bid too low. Current highest bid is: " + strconv.Itoa(int(highestBid)))
	} else {
		r.CurrentBids[req.BidderId] = req.Amount
	}

	response := &proto.Response{
		RequestId: req.RequestId,
		Message:   message,
		Success:   success,
	}

	// Save the response for later retrieval
	r.Responses[req.RequestId] = response
	return response, nil
}

func (r *ReplicaManager) Bid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if time.Now().After(r.AuctionEnd) {
		fmt.Println("Bid rejected: Auction has ended.")
		return &proto.BidResponse{Outcome: "Auction has ended. Bids are no longer accepted."}, nil
	}

	currentHighestBid, exists := r.CurrentBids[req.BidderId]
	if !exists {
		fmt.Printf("New bidder registered: %s\n", req.BidderId)
	}
	if !exists || req.Amount > currentHighestBid {
		r.CurrentBids[req.BidderId] = req.Amount
		fmt.Printf("Bid accepted from %s: %d\n", req.BidderId, req.Amount)
		return &proto.BidResponse{Outcome: "Bid accepted."}, nil
	}

	fmt.Println("Bid rejected: Bid too low.")
	return &proto.BidResponse{Outcome: "Bid too low. Please place a higher bid."}, nil
}

func (r *ReplicaManager) Result(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// Check if the auction has ended
	if time.Now().After(r.AuctionEnd) {
		// Auction has ended, find the highest bid
		var winningBid int32
		var winner string
		for bidderId, bidAmount := range r.CurrentBids {
			if bidAmount > winningBid {
				winningBid = bidAmount
				winner = bidderId
			}
		}

		// Return the result with the winner's information
		if winner != "" {
			return &proto.ResultResponse{Outcome: fmt.Sprintf("Auction ended. Winner: %s with bid: %d", winner, winningBid)}, nil
		} else {
			return &proto.ResultResponse{Outcome: "Auction ended with no bids."}, nil
		}
	}

	// Auction is still ongoing, return a message indicating this
	return &proto.ResultResponse{Outcome: "Auction is still ongoing. Result not available yet."}, nil
}

func (r *ReplicaManager) HandleLeadershipTransfer(ctx context.Context, req *proto.LeadershipTransferRequest) (*proto.LeadershipTransferResponse, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.isCurrentlyLeader = true
	r.AuctionEnd = time.Unix(req.AuctionEndTime, 0) // Set auction end time
	fmt.Println("Leadership transferred to this Replica Manager with Auction End Time:", r.AuctionEnd)
	return &proto.LeadershipTransferResponse{}, nil
}

func (r *ReplicaManager) CheckHealth(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	// Implement your health check logic here.
	// This is a basic implementation, always returning that the server is healthy.
	// You can expand this to perform actual health checks as needed.
	return &proto.HealthCheckResponse{Healthy: true}, nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: replica_manager <port>")
	}

	port := os.Args[1]

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	replicaManagerServer := grpc.NewServer()
	replicaManager := NewReplicaManager()

	proto.RegisterAuctionServer(replicaManagerServer, replicaManager)

	log.Printf("Replica Manager created. Listening on port %s", lis.Addr().String())

	if err := replicaManagerServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
