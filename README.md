# Solana-Wallet-Sniper

This is a fun project written in Go that helps you find Solana vanity addresses! It's designed to be super fast; when running on a good server, it can check around 480,000 keys per second!

**THIS PROGRAM IS CPU BASED:** if you are looking at getting more speed but a harder setup find another repo that supports gpu mining for wallet generation.

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

This project was originally made with `go1.20.1`. It's highly recommended to use the [Pre-built executable](https://github.com/LiteEagle262/Solana-Wallet-Sniper/releases/tag/v1) if your version does not work with the program.

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

Example Finds: 

![image](https://github.com/user-attachments/assets/e98f9f79-1f2f-45a0-94d3-6cb7b9949d17)

![image](https://github.com/user-attachments/assets/dee7160c-0621-4c8f-9845-bfe84954bd6b)

