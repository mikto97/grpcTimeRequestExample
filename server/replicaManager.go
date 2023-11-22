package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	proto "grpc/grpc"

	"google.golang.org/grpc"
)

type ReplicaManager struct {
	Responses map[string]*proto.Response // map[requestID]*Response
	proto.UnimplementedAuctionServer
}

func NewReplicaManager() *ReplicaManager {
	return &ReplicaManager{
		Responses: make(map[string]*proto.Response),
	}
}

func (r *ReplicaManager) ReplicateBidRequest(ctx context.Context, req *proto.BidRequest) (*proto.Response, error) {
	// Replica manager logic to handle replicated bid requests
	response := &proto.Response{
		RequestId: req.RequestId,
		Success:   true, // Set based on the success of replication

	}
	// Simulate a delay to mimic the time taken for coordination and execution
	time.Sleep(500 * time.Millisecond)

	// Save the response for later retrieval by the main server
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
