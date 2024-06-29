# Load Balancer 🚀

This project implements an advanced HTTP load balancer in Go written according to a specific use case (randomized round-robin). The load balancer forwards incoming HTTP requests to a list of primary RPC (Remote Procedure Call) endpoints. If all primary endpoints fail, it falls back to a secondary list of RPC endpoints. Additionally, it includes a caching mechanism to temporarily avoid failed endpoints and a notification system to alert when all endpoints are down. 📡

## Architecture 🏗️
![IMAGE](architecture.jpg)

## Logic Overview 🧠

1. **Initialization**:
   - The `.env` file is loaded to fetch the primary and fallback RPC URLs.
   - The URLs are split into slices and stored in global variables.
   - The application fails to start if no primary or fallback RPCs are provided. ❌

2. **Main Function**:
   - Initializes a cache to store failed URLs temporarily using an in-memory database. 🗃️
   - Starts an HTTP server that listens for incoming requests. 🌐

3. **Load Balancer**:
   - Handles incoming HTTP requests and attempts to forward them to the primary RPCs. 📬
   - If all primary RPCs fail, it tries the fallback RPCs. 🔄
   - If both primary and fallback RPCs fail, it sends a notification to Slack and returns an error response. 🚨

4. **Request Forwarding**:
   - Shuffles the list of URLs to distribute the load. 🔀
   - Forwards the request to each URL until a successful response is received. ✅
   - If a URL returns a "Too Many Requests" status, it is cached in a Two-Queue (2Q) in-memory database to avoid retrying it for a specified TTL (time-to-live). ⏳

5. **Caching Mechanism**:
   - Uses a Two-Queue (2Q) in-memory caching algorithm to store failed URLs. 🗄️
   - The cache temporarily stores URLs that fail to respond or return a "Too Many Requests" status, preventing repeated attempts to the same failing endpoint. 🚫
   - `ERROR_TIME_TO_LIVE_MINUTES` is the time-to-live (TTL) for each URL in the cache. ⏲️

6. **Slack Notification**:
   - Sends a notification to Slack when all RPC endpoints fail to `SLACK_WEBHOOK_URL`. 📩

## Installation 🛠️

1. **Clone the repository**:
   ```sh
   git clone https://github.com/Diffusion-Labs/load-balancer
   cd load-balancer
   ```

2. **Install dependencies**:
   Ensure you have Go installed. Then, run:
   ```sh
   go mod tidy
   ```

3. **Set up environment variables**:
   Create a `.env` file in the root directory with the following content:
   ```env
   RPCs="https://rpc1.example.com, https://rpc2.example.com"
   FALLBACK_RPCs="https://rpc.mantle.xyz"
   PORT=8080
   SLACK_WEBHOOK_URL=""
   ERROR_TIME_TO_LIVE_MINUTES=3
   ```

## Usage 🚀

1. **Run the load balancer**:
   ```sh
   go run main.go
   ```

2. **Send HTTP requests**:
   The load balancer listens on `http://localhost:<ENV:PORT>`. You can send HTTP requests to this address, and the load balancer will forward them to the configured RPC endpoints. 📡

3. **Monitor logs**:
   The application logs the status of each request and any errors encountered. Check the console output for real-time logs. 📋

## Example

To test the load balancer, you can use `curl http://localhost:8080` to send an HTTP request to the load balancer. The load balancer will forward the request to the primary and fallback RPC endpoints based on the configuration. 🧪
