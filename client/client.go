package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	proto "grpc/grpc"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var replicaManagerAddresses = []string{"localhost:50040", "localhost:50041", "localhost:50042"}

func main() {

	flag.Parse()

	BidderID := flag.Arg(0)
	port, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		log.Fatal("Invalid port number:", err)
	}

	// Dial to the gRPC server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:50051"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Create a client instance
	client := proto.NewAuctionClient(conn)

	fmt.Printf("%s connected at port: %d\n", BidderID, port)

	// Allow users to bid until the predefined time period ends
	fmt.Printf("Auction is open. You can bid until %s.\n", time.Now().Add(100*time.Second).Format("15:04:05"))

	for {
		// Generate a unique request identifier
		requestID := generateUniqueRequestID()

		// Get user input for the bid amount
		var bidAmount int
		fmt.Print("Enter your bid amount (0 to exit): ")
		fmt.Scan(&bidAmount)

		if bidAmount == 0 {
			break
		}

		// Create a bid request with the unique identifier
		bidRequest := &proto.BidRequest{
			RequestId: requestID,
			BidderId:  BidderID,
			Amount:    int32(bidAmount),
		}

		// Multicast the bid request to all replica managers
		sendBidRequestToReplicaManagers(client, bidRequest)

		// Wait for responses from all replica managers
		responses := waitForResponsesFromReplicaManagers(client)

		// Process responses
		processResponses(responses)
	}

	// Display the result at the end
	resultRequest := &proto.ResultRequest{
		RequestId: generateUniqueRequestID(),
	}
	resultResponse, err := client.Result(context.Background(), resultRequest)
	if err != nil {
		log.Fatalf("Result failed: %v", err)
	}
	fmt.Println("Result Outcome:", resultResponse.Outcome)
}

func generateUniqueRequestID() string {
	return uuid.New().String()
}

func sendBidRequestToReplicaManagers(client proto.AuctionClient, request *proto.BidRequest) {
	// Iterate over replica managers and send bid request
	for _, replicaManagerAddress := range replicaManagerAddresses {
		conn, err := grpc.Dial(replicaManagerAddress, grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect to replica manager %s: %v", replicaManagerAddress, err)
			continue
		}
		defer conn.Close()

		// Create a client instance for the replica manager
		client := proto.NewAuctionClient(conn)

		// Send the bid request to the replica manager
		_, err = client.ReplicateBidRequest(context.Background(), request)
		if err != nil {
			log.Printf("Failed to replicate bid to %s: %v", replicaManagerAddress, err)
			// Handle error as needed
		}
	}
}

func waitForResponsesFromReplicaManagers(client proto.AuctionClient) []*proto.Response {
	empty := &proto.Empty{}
	response, err := client.WaitForResponsesFromReplicaManagers(context.Background(), empty)
	if err != nil {
		log.Printf("Failed to receive responses: %v", err)
		// Handle error as needed
		return nil
	}
	return response.Responses
}
func processResponses(responses []*proto.Response) {
	for _, response := range responses {
		// Process each response
		if response.Success {
			fmt.Printf("Replication succeeded for request ID %s\n", response.RequestId)
		} else {
			fmt.Printf("Replication failed for request ID %s. Error: %s\n", response.RequestId, response.Error)
		}
	}
}
