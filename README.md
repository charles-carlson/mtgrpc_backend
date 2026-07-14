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

## Milestones
- [x] Connect / gRPC / gRPC-Web service, 7 RPCs (Add, Get, GetByName, GetBySet, Search, List, ListSets)
- [x] DynamoDB store with a `set-index` GSI; Scryfall enrichment (image, colors, rarity, prices)
- [x] Private ingest/eject tooling (`.json` / `.txt` / `.csv`) run via CLI flags at startup
- [ ] In-memory read model: one `ScanAll` at boot into an immutable snapshot, served for List/Search (Dynamo stays the source of truth, rebuilt each restart)
- [ ] Price refresh via Scryfall **bulk data** (`default_cards`) instead of per-card calls
- [ ] Colorless filter support (match an empty `colors` array)
- [ ] Multi-set filter (`repeated string sets` in the proto)
- [ ] Optional: live refresh path (admin RPC / SIGHUP → re-scan → atomic swap) so ingest doesn't require a restart

## Potential Fixes / Known Issues
- **`RefreshPrices` is O(n) sequential Scryfall calls.** Gated at 10 req/s → ~8 min at 5k cards, ~25+ min at 15k, with no checkpoint (a restart loses progress). Replace with a single `default_cards` bulk download + local join, and batch the writes. Bounded worker concurrency is a cheaper interim step.
- **`Search` without a set does a full-table `Scan` + `FilterExpression`.** DynamoDB's `Limit` caps items *examined*, not returned — so a filtered page can come back near-empty while still emitting a `next_page_token`. Pagination gets visibly worse as the collection grows. Fix via the in-memory read model, or loop `Scan` pages until `pageSize` post-filter results accumulate.
- **`QueryBySet` looks broken.** It runs a `Query` on `set-index` with the set match in `FilterExpression` and **no `KeyConditionExpression`** — a `Query` requires one, so this should error at runtime. `Search`'s set branch does it correctly (`KeyConditionExpression: #s = :set`); mirror that and confirm `GetCardsBySet` actually returns data.
- **Colorless filtering has no backend path.** Scryfall stores `colors: []` for colorless cards (the `C` symbol only appears in `produced_mana` / mana costs, not `colors`). Supporting it needs a sentinel + a `size(#colors) = 0` condition in `store.Search`.
- **`SearchCardsRequest.set` is a single string.** Clients are limited to single-set filtering; multi-set needs `repeated string sets` + regen + `Search` handling.
