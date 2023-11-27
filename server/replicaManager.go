package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	proto "grpc/grpc"

	"google.golang.org/grpc"
)

type ReplicaManager struct {
	Responses         map[string]*proto.Response // map[requestID]*Response
	CurrentBids       map[string]int32           // map[bidderID]bidAmount
	isCurrentlyLeader bool                       // Simplified leadership logic
	Mutex             sync.Mutex                 // Mutex to protect shared resources
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

	if req.Amount <= highestBid {
		message = "Bid too low. Current highest bid is: " + strconv.Itoa(int(highestBid))
		success = false
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
