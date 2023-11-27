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
	LeaderLease       time.Time                  // Lease expiration time for leader
	LeaseDuration     time.Duration              // Duration for which a lease is valid
	isCurrentlyLeader bool                       // Flag to indicate if this replica is the leader
	Mutex             sync.Mutex                 // Mutex to protect shared resources
	proto.UnimplementedAuctionServer
}

// NewReplicaManager initializes a new ReplicaManager
func NewReplicaManager(leaseDuration time.Duration) *ReplicaManager {
	return &ReplicaManager{
		Responses:         make(map[string]*proto.Response),
		CurrentBids:       make(map[string]int32),
		LeaderLease:       time.Now().Add(leaseDuration),
		LeaseDuration:     leaseDuration,
		isCurrentlyLeader: false,
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

func (r *ReplicaManager) WaitForResponses(ctx context.Context, req *proto.Empty) (*proto.RepeatedResponse, error) {
	// Simulate waiting for responses from all replica managers
	time.Sleep(2 * time.Second)

	// Collect and return responses
	responses := make([]*proto.Response, 0, len(r.Responses))
	for _, response := range r.Responses {
		responses = append(responses, response)
	}

	return &proto.RepeatedResponse{
		Responses: responses,
	}, nil
}
func (r *ReplicaManager) tryAcquireLease() bool {
	// Simulate acquiring the lease by setting the lease expiration time
	r.LeaderLease = time.Now().Add(r.LeaseDuration)

	return true
}

func (r *ReplicaManager) sendHeartbeat() {
	for {
		// Implement logic to send a heartbeat to the main server
		// For simplicity, let's use a sleep to simulate heartbeat
		time.Sleep(r.LeaseDuration / 2)

		// If the main server is not responding, initiate leader election

		r.tryAcquireLease()

	}
}

func (r *ReplicaManager) HandleLeadershipTransfer(ctx context.Context, req *proto.LeadershipTransferRequest) (*proto.LeadershipTransferResponse, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// The Replica Manager becomes the leader
	r.isCurrentlyLeader = true

	// Log or handle additional logic as necessary
	log.Println("This replica manager is now the leader.")

	return &proto.LeadershipTransferResponse{Success: true}, nil
}

func (r *ReplicaManager) startLeaderElection() {
	fmt.Println("StartedLeaderElection")
	// Start a goroutine for leader election
	go func() {
		for {
			// Try to acquire the lease
			if r.tryAcquireLease() {
				r.isCurrentlyLeader = true
				fmt.Println("I am the new leader!")
			} else {
				r.isCurrentlyLeader = false
			}

			// Wait for a short duration before attempting to acquire the lease again
			time.Sleep(r.LeaseDuration / 2)
		}
	}()
}

// Implement checkLease in ReplicaManager to check if the lease is currently held
func (r *ReplicaManager) checkLease() bool {
	// Simulate checking if the lease is currently held
	return time.Now().Before(r.LeaderLease)
}

// Implement acquireLease in ReplicaManager to acquire the lease
func (r *ReplicaManager) acquireLease() bool {
	// Simulate acquiring the lease by setting the lease expiration time
	r.LeaderLease = time.Now().Add(r.LeaseDuration)
	return false
}

func (r *ReplicaManager) CheckHealth(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	// Implement your health check logic here.
	// This is a basic implementation, always returning that the server is healthy.
	// You can expand this to perform actual health checks as needed.
	return &proto.HealthCheckResponse{Healthy: true}, nil
}

func (r *ReplicaManager) WaitForResponsesFromReplicaManagers(ctx context.Context, req *proto.Empty) (*proto.Responses, error) {
	// Simulate waiting for responses from all replica managers
	time.Sleep(2 * time.Second)

	// Collect and return responses
	responses := make([]*proto.Response, 0, len(r.Responses))
	for _, response := range r.Responses {
		responses = append(responses, response)
	}

	return &proto.Responses{
		Responses: responses,
	}, nil
}
func (r *ReplicaManager) IsLeader(ctx context.Context, req *proto.LeaderCheckRequest) (*proto.LeaderCheckResponse, error) {
	return &proto.LeaderCheckResponse{IsLeader: r.isCurrentlyLeader}, nil
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
	leaseDuration := 10 * time.Second

	replicaManagerServer := grpc.NewServer()
	replicaManager := NewReplicaManager(leaseDuration)

	proto.RegisterAuctionServer(replicaManagerServer, replicaManager)

	// Start leader election and heartbeat
	//replicaManager.startLeaderElection()
	go replicaManager.sendHeartbeat()

	log.Printf("Replica Manager created. Listening on port %s", lis.Addr().String())

	if err := replicaManagerServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
