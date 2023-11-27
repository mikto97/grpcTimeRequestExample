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

func GetLeaderAddress() string {
	allServerAddresses := []string{"localhost:50051", "localhost:50040"}

	for _, address := range allServerAddresses {
		conn, err := grpc.Dial(address, grpc.WithInsecure()) // Adjust for security as necessary
		if err != nil {
			log.Printf("Error connecting to server at %s: %v", address, err)
			continue
		}
		defer conn.Close()

		client := proto.NewAuctionClient(conn)
		resp, err := client.IsLeader(context.Background(), &proto.LeaderCheckRequest{})
		if err == nil && resp.IsLeader {
			log.Printf("Current leader found at %s", address)
			return address // This server is the current leader
		}
	}

	log.Println("No leader found among the servers")
	return "" // Returning empty string, but consider handling this case differently
}

func main() {
	flag.Parse()

	BidderID := flag.Arg(0)
	port, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		log.Fatal("Invalid port number:", err)
	}

	leaderAddress := GetLeaderAddress()
	if leaderAddress == "" {
		log.Fatal("No leader address available, exiting")
	}
	// Dial to the gRPC server
	conn, err := grpc.Dial(leaderAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Create a client instance
	client := proto.NewAuctionClient(conn)

	fmt.Printf("%s connected at port: %d\n", BidderID, port)

	// Allow users to bid until the predefined time period ends
	fmt.Printf("Auction is open. You can bid until %s.\n", time.Now().Add(30*time.Second).Format("15:04:05"))

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
		fmt.Println("Multicasting bid request to replica managers...")
		sendBidRequestToReplicaManagers(client, bidRequest)
		if err != nil {
			fmt.Println("Error sending bid request, retrying...")
			time.Sleep(time.Second * 1) // Wait for 1 second before retrying
			continue
		}

		// Wait for responses from all replica managers
		fmt.Println("Waiting for responses from replica managers...")
		responses := waitForResponsesFromReplicaManagers(client)
		fmt.Printf("Received %d responses from replica managers.\n", len(responses))

		// Process responses
		fmt.Println("Processing responses...")
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

func sendBidRequestToReplicaManagers(client proto.AuctionClient, request *proto.BidRequest) error {
	var lastErr error

	for retry := 0; retry < 3; retry++ { // Retry the whole process up to 3 times
		// Get the latest list of replica manager addresses
		replicaManagerAddresses := GetReplicaManagerAddresses()

		for _, replicaManagerAddress := range replicaManagerAddresses {
			success := false

			for attempt := 0; attempt < 3; attempt++ { // Retry connecting to each replica manager up to 3 times
				conn, err := grpc.Dial(replicaManagerAddress, grpc.WithInsecure())
				if err != nil {
					log.Printf("Attempt %d: failed to connect to replica manager %s: %v", attempt+1, replicaManagerAddress, err)
					lastErr = err
					time.Sleep(time.Second) // Wait for 1 second before retrying
					continue
				}

				replicaManagerClient := proto.NewAuctionClient(conn)

				_, err = replicaManagerClient.ReplicateBidRequest(context.Background(), request)
				conn.Close() // Close the connection whether it succeeds or fails

				if err != nil {
					log.Printf("Attempt %d: failed to replicate bid to %s: %v", attempt+1, replicaManagerAddress, err)
					lastErr = err
					time.Sleep(time.Second) // Wait for 1 second before retrying
				} else {
					fmt.Printf("Successfully replicated bid to %s.\n", replicaManagerAddress)
					success = true
					break // Break out of the retry loop on success
				}
			}

			if !success {
				log.Printf("Failed to replicate bid to replica manager %s after multiple attempts", replicaManagerAddress)
				// Optionally, you can choose to continue to the next replica manager or return an error
			}
		}

		if lastErr == nil {
			return nil // Successfully sent request to at least one replica manager
		}

		// If all attempts failed, wait for a moment before retrying the whole process
		log.Printf("Retrying the entire replication process...")
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("failed to replicate bid to any replica managers after multiple retries: %v", lastErr)
}

func GetReplicaManagerAddresses() []string {
	// Logic to retrieve the latest list of replica manager addresses
	// This could be from a configuration service, file, etc.
	// For simplicity, returning a hardcoded list here
	return []string{"localhost:50040"}
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
		if response.Success {
			fmt.Printf("Response from %s: %s\n", response.RequestId, response.Message)
		} else {
			fmt.Printf("Replication failed: %s\n", response.Error)
		}
	}
}
