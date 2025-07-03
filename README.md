# Solana-Wallet-Sniper

This is a fun project written in Go that helps you find Solana vanity addresses! It's designed to be super fast; when running on a good server, it can check around 480,000 keys per second!

The time it takes to find an address with a specific prefix depends on how many characters are in that prefix. The probabilities and estimated times (based on 480,000 keys/second) are:

* **2 Characters:** $(1/58)^2 \approx 1 \text{ in } 3,364$
  * **Estimated Time:** $\approx 0.007 \text{ seconds}$
* **3 Characters:** $(1/58)^3 \approx 1 \text{ in } 195,112$
  * **Estimated Time:** $\approx 0.41 \text{ seconds}$
* **4 Characters:** $(1/58)^4 \approx 1 \text{ in } 11,316,496$
  * **Estimated Time:** $\approx 23.58 \text{ seconds}$
* **5 Characters:** $(1/58)^5 \approx 1 \text{ in } 656,356,768$
  * **Estimated Time:** $\approx 22.7 \text{ minutes}$
* **6 Characters:** $(1/58)^6 \approx 1 \text{ in } 38,068,692,544$
  * **Estimated Time:** $\approx 1.84 \text{ days}$
* **7 Characters:** $(1/58)^7 \approx 1 \text{ in } 2,207,984,167,552$
  * **Estimated Time:** $\approx 53.2 \text{ days}$

This project was originally made with `go1.20.1`. It's highly recommended to use the latest stable Go release or a pre-built binary for the best performance and security.

### How to Use

The project is simple to use and includes an optional Discord notification system.

**If you are running the release build (e.g., `wallet_snipe.exe` on Windows):**

```bash
wallet_snipe.exe eagle https://discord.com/api/webhooks/your-webhook-url
```

**If you are running from the source code:**
```
go run main.go eagle https://discord.com/api/webhooks/your-webhook-url
```

The **Discord Webhook** argument is **optional**. If you don't provide it, the results will still be printed to your console.

