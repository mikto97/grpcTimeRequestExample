Distributed Auction System
This Distributed Auction System allows clients to participate in an auction by placing bids. The system uses a Replica Manager for fault tolerance, capable of taking over in case the main server fails.

System Components
Main Server: Handles the auction logic including accepting bids and determining the auction winner.
Replica Manager: Acts as a backup server, replicating bids and capable of taking over from the Main Server.
Client: Used by bidders to place bids and query auction results.
Prerequisites
Go programming language
gRPC and Protocol Buffers
Setup and Running
Start the Main Server:
Navigate to the server directory and run the following command:

go run server.go
This will start the auction server listening for client bids and managing the auction.

Start the Replica Manager:
Open another terminal and navigate to the replica manager directory. Start the replica manager using:


go run replicaManager.go :50040


Using the Client:
To place bids or query the auction, use the client program. Navigate to the client directory and run:

go run client.go <BidderID> <Port>
Replace <BidderID> with your unique identifier and <Port> with the port number.

How to Use the Client
Place a Bid:
When prompted, enter your bid amount. The bid must be higher than your previous bid.

Query Auction Result:
The client will automatically query and display the current highest bid before each bidding opportunity.

Exit:
To exit the bidding process, enter 0 when prompted for a bid amount.

Auction Rules
The auction runs for a predefined time limit (e.g., 100 time units from the start of the system).
The first call to "bid" registers the bidder.
Bidders can bid several times, but each subsequent bid must be higher than the previous one.
When the auction time ends, the highest bidder is declared the winner.
Bidders can query the system at any time to know the state of the auction.
Fault Tolerance
The system is designed to be resilient to the failure of one node (Main Server). In such an event, the Replica Manager takes over the responsibilities of the Main Server, ensuring the continuity of the auction.