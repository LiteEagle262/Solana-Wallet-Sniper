package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gagliardetto/solana-go"
	"github.com/tyler-smith/go-bip39"
	"github.com/mr-tron/base58"
)

type DiscordPayload struct {
	Embeds []Embed `json:"embeds"`
}

type Embed struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Color       int     `json:"color"`
	Fields      []Field `json:"fields"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func fastPrefixCheck(pubKeyBytes []byte, targetBytes []byte) bool {
	if len(targetBytes) == 0 {
		return true
	}
	
	encoded := base58.Encode(pubKeyBytes)
	return len(encoded) >= len(targetBytes) && 
		   encoded[0] == targetBytes[0] && 
		   (len(targetBytes) == 1 || encoded[1] == targetBytes[1])
}

func generateOptimizedKey() (ed25519.PrivateKey, ed25519.PublicKey) {
	seed := make([]byte, 32)
	rand.Read(seed)
	
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	
	return privateKey, publicKey
}

func generateMnemonicFromPrivateKey(privateKey ed25519.PrivateKey) string {
	seed := privateKey.Seed()
	mnemonic, _ := bip39.NewMnemonic(seed)
	return mnemonic
}

func main() {
	fmt.Println("🚀 Starting Solana Vanity Address Generator...")
	
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <TARGET_PREFIX> [DISCORD_WEBHOOK_URL]")
		fmt.Println("Example: go run main.go Sol https://discord.com/api/webhooks/...")
		fmt.Println("Example: go run main.go Sol (without Discord notification)")
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
	
	fmt.Printf("✅ Arguments parsed successfully\n")
	
	targetPrefix := os.Args[1]
	var webhookURL string
	if len(os.Args) >= 3 {
		webhookURL = os.Args[2]
	}
	
	fmt.Printf("🎯 Target prefix: %s\n", targetPrefix)
	if webhookURL != "" {
		fmt.Printf("📤 Discord webhook: configured\n")
	} else {
		fmt.Printf("📤 Discord webhook: not configured\n")
	}
	
	targetBytes := []byte(targetPrefix)

	var wg sync.WaitGroup
	found := make(chan struct{})
	startTime := time.Now()
	var generatedCount uint64 = 0

	numWorkers := runtime.NumCPU() * 2
	fmt.Printf("🚀 TURBO MODE: Starting %d workers to find public key starting with '%s'...\n", numWorkers, targetPrefix)
	fmt.Printf("💡 Optimizations: Direct key generation, fast prefix check, minimal allocations\n\n")
	
	time.Sleep(1 * time.Second)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			var privateKey ed25519.PrivateKey
			var publicKey ed25519.PublicKey
			
			for {
				select {
				case <-found:
					return
				default:
					privateKey, publicKey = generateOptimizedKey()
					
					if fastPrefixCheck(publicKey, targetBytes) {
						solanaPrivateKey := solana.PrivateKey(privateKey)
						solanaPublicKey := solanaPrivateKey.PublicKey()
						
						if strings.HasPrefix(solanaPublicKey.String(), targetPrefix) {
							select {
							case <-found:
								return
							default:
								close(found)
								elapsedTime := time.Since(startTime)
								
								mnemonic := generateMnemonicFromPrivateKey(privateKey)

								fmt.Printf("\n\n🎉 JACKPOT! Wallet found in %s!\n", elapsedTime.Round(time.Millisecond))
								fmt.Println("========================================")
								fmt.Printf("🔑 Public Key: %s\n", solanaPublicKey)
								fmt.Printf("🔐 Private Key: %s\n", solanaPrivateKey)
								fmt.Printf("📝 Mnemonic: %s\n", mnemonic)
								fmt.Printf("⚡ Keys checked: %d\n", atomic.LoadUint64(&generatedCount))
								fmt.Printf("🏃 Average rate: %.0f keys/sec\n", float64(atomic.LoadUint64(&generatedCount))/elapsedTime.Seconds())
								fmt.Println("========================================")

								if webhookURL != "" {
									fmt.Println("📤 Sending to Discord...")
									sendToDiscord(webhookURL, solanaPublicKey.String(), mnemonic, elapsedTime)
								}
							}
							return
						}
					}
					
					if atomic.AddUint64(&generatedCount, 1) % 1000 == 0 {
						runtime.Gosched()
					}
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		lastCount := uint64(0)
		lastTime := time.Now()

		for {
			select {
			case <-found:
				return
			case <-ticker.C:
				now := time.Now()
				currentCount := atomic.LoadUint64(&generatedCount)
				deltaCount := currentCount - lastCount
				deltaTime := now.Sub(lastTime).Seconds()
				
				rate := float64(deltaCount) / deltaTime
				lastCount = currentCount
				lastTime = now
				
				fmt.Printf("\r🔍 Searching... | 🚀 Rate: %.0f keys/sec | 📈 Total: %d | ⏱️  %v        ", 
					rate, currentCount, time.Since(startTime).Round(time.Second))
			}
		}
	}()

	wg.Wait()
}

func sendToDiscord(webhookURL, publicKey, mnemonic string, duration time.Duration) {
	payload := DiscordPayload{
		Embeds: []Embed{
			{
				Title:       "🎉 Solana Vanity Address Found!",
				Description: "A new wallet matching the criteria has been generated.",
				Color:       3066993,
				Fields: []Field{
					{
						Name:   "Public Key",
						Value:  fmt.Sprintf("`%s`", publicKey),
						Inline: false,
					},
					{
						Name:   "Mnemonic Phrase (Private)",
						Value:  fmt.Sprintf("||`%s`||", mnemonic),
						Inline: false,
					},
					{
						Name:   "Time to Find",
						Value:  duration.Round(time.Second).String(),
						Inline: true,
					},
					{
						Name:   "Timestamp",
						Value:  time.Now().Format(time.RFC1123),
						Inline: true,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling Discord payload:", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error sending to Discord:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("Discord webhook returned non-success status: %s\n", resp.Status)
	}
}
