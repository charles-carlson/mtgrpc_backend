# MTG-GRPC
Backend Application to my personal Virtualized Binder of MTG Cards. Serves necessary information for a client side component and internal tooling for ingesting collection.
## Idea
I wanted to store my cards virtually, as a means to keep track of what I own. It will be able to provide myself and others information on liquid value, set completion, and the amount of cards I own. I decided grpc as it can easily handle batches of file data or record data. Type safe queries with known requests and responses makes data handling more predictable.
## Structure
Cards follow a simple structure masking the proto skeleton that has been set up. I keep internal tools private from the actual services defined in my proto as I am not allowing anyone but myself the ability to add and remove cards on mass. I use scryfall's generous request limit to gather card data. My program follows a service pattern to encapsulate grpc handling, service handling, and dynamo handling completely separate from each other. 
## Implementation
Implmented in Golang & Proto alongside Terraform. I use Go's testing suite to test service functions and private tooling (Ingest, Eject).
## Cloud Resources
EC2, ECR, IAM, CloudWatch, Network Load Balancer, and DynamoDB. I manage my application's docker image in ECR, and use it as base image for my EC2 instance. Cloudwatch handles logging, through an interceptor, and since I am not using http/1, I require a network load balancer for my application. IAM controls access between ECR, EC2, and dynamoDB. I went with DynamoDB as I am keep records simple as possible, with easy upserts and key-queries for nosql pagination.
