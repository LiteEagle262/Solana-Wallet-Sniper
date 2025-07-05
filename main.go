package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/tyler-smith/go-bip39"
	"github.com/mr-tron/base58"
)

type DiscordPayload struct {
	Embeds []Embed `json:"embeds"`
}

type Embed struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Color       int     `json:"color"`
	Fields      []Field `json:"fields"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type WalletResult struct {
	mnemonic   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
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

// Proper BIP32 HMAC-SHA512 based key derivation
func deriveKey(parentKey, parentChainCode []byte, index uint32) ([]byte, []byte) {
	data := make([]byte, 37)
	data[0] = 0x00 // Private key derivation
	copy(data[1:33], parentKey)
	binary.BigEndian.PutUint32(data[33:], index)
	
	mac := hmac.New(sha512.New, parentChainCode)
	mac.Write(data)
	hash := mac.Sum(nil)
	
	return hash[:32], hash[32:]
}

// Proper BIP44 derivation for Solana: m/44'/501'/0'/0'
func deriveSolanaKeyFromSeed(seed []byte) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	// Generate master key from seed
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	masterHash := mac.Sum(nil)
	
	masterKey := masterHash[:32]
	masterChainCode := masterHash[32:]
	
	// BIP44 path: m/44'/501'/0'/0'
	derivationPath := []uint32{
		44 + 0x80000000,  // Purpose (hardened)
		501 + 0x80000000, // Coin type - Solana (hardened)  
		0 + 0x80000000,   // Account (hardened)
		0 + 0x80000000,   // Change (hardened)
	}
	
	currentKey := masterKey
	currentChainCode := masterChainCode
	
	for _, index := range derivationPath {
		currentKey, currentChainCode = deriveKey(currentKey, currentChainCode, index)
	}
	
	// Create Ed25519 key pair
	privateKey := ed25519.NewKeyFromSeed(currentKey)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	
	return privateKey, publicKey, nil
}

func generateOptimizedWallet() (WalletResult, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return WalletResult{}, err
	}
	
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return WalletResult{}, err
	}
	
	// Generate seed from mnemonic (with empty passphrase)
	seed := bip39.NewSeed(mnemonic, "")
	
	privateKey, publicKey, err := deriveSolanaKeyFromSeed(seed)
	if err != nil {
		return WalletResult{}, err
	}
	
	return WalletResult{
		mnemonic:   mnemonic,
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

type EntropyPool struct {
	pool chan []byte
	mu   sync.Mutex
}

func NewEntropyPool(size int) *EntropyPool {
	pool := make(chan []byte, size)
	
	go func() {
		for {
			entropy := make([]byte, 32)
			if _, err := rand.Read(entropy); err != nil {
				time.Sleep(time.Millisecond)
				continue
			}
			select {
			case pool <- entropy:
			default:
				time.Sleep(time.Microsecond)
			}
		}
	}()
	
	return &EntropyPool{pool: pool}
}

func (ep *EntropyPool) GetEntropy() []byte {
	select {
	case entropy := <-ep.pool:
		return entropy
	default:
		entropy := make([]byte, 32)
		rand.Read(entropy)
		return entropy
	}
}

func generateTurboWallet(entropyPool *EntropyPool) (WalletResult, error) {
	entropy := entropyPool.GetEntropy()
	
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return WalletResult{}, err
	}
	
	seed := bip39.NewSeed(mnemonic, "")
	
	privateKey, publicKey, err := deriveSolanaKeyFromSeed(seed)
	if err != nil {
		return WalletResult{}, err
	}
	
	return WalletResult{
		mnemonic:   mnemonic,
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
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

	entropyPool := NewEntropyPool(1000)
	
	var wg sync.WaitGroup
	found := make(chan WalletResult, 1)
	startTime := time.Now()
	var generatedCount uint64 = 0

	numWorkers := runtime.NumCPU() * 4
	fmt.Printf("🚀 Starting %d workers to find public key starting with '%s'...\n", numWorkers, targetPrefix)
	
	time.Sleep(1 * time.Second)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for {
				select {
				case <-found:
					return
				default:
					wallet, err := generateTurboWallet(entropyPool)
					if err != nil {
						continue
					}
					
					if fastPrefixCheck(wallet.publicKey, targetBytes) {
						solanaPrivateKey := solana.PrivateKey(wallet.privateKey)
						solanaPublicKey := solanaPrivateKey.PublicKey()
						
						if strings.HasPrefix(solanaPublicKey.String(), targetPrefix) {
							select {
							case found <- wallet:
								return
							default:
								return
							}
						}
					}
					
					count := atomic.AddUint64(&generatedCount, 1)
					if count%500 == 0 {
						runtime.Gosched()
					}
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
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
				
				fmt.Printf("\r🔍 Searching... | 🚀 Rate: %.0f keys/sec | 📈 Total: %d | ⏱️  %v        ", 
					rate, currentCount, time.Since(startTime).Round(time.Second))
			}
		}
	}()

	wallet := <-found
	elapsedTime := time.Since(startTime)
	
	solanaPrivateKey := solana.PrivateKey(wallet.privateKey)
	solanaPublicKey := solanaPrivateKey.PublicKey()

	fmt.Printf("\n\n🎉 JACKPOT! Wallet found in %s!\n", elapsedTime.Round(time.Millisecond))
	fmt.Println("========================================")
	fmt.Printf("🔑 Public Key: %s\n", solanaPublicKey)
	fmt.Printf("🔐 Private Key: %s\n", solanaPrivateKey)
	fmt.Printf("📝 Mnemonic: %s\n", wallet.mnemonic)
	fmt.Printf("⚡ Keys checked: %d\n", atomic.LoadUint64(&generatedCount))
	fmt.Printf("🏃 Average rate: %.0f keys/sec\n", float64(atomic.LoadUint64(&generatedCount))/elapsedTime.Seconds())
	fmt.Println("========================================")

	if webhookURL != "" {
		fmt.Println("📤 Sending to Discord...")
		sendToDiscord(webhookURL, solanaPublicKey.String(), wallet.mnemonic, elapsedTime)
	}

	wg.Wait()
}

func sendToDiscord(webhookURL, publicKey, mnemonic string, duration time.Duration) {
	payload := DiscordPayload{
		Embeds: []Embed{
			{
				Title:       "🎉 Solana Vanity Address Found!",
				Description: "A new wallet matching the criteria has been generated.",
				Color:       3066993,
				Fields: []Field{
					{
						Name:   "Public Key",
						Value:  fmt.Sprintf("`%s`", publicKey),
						Inline: false,
					},
					{
						Name:   "Mnemonic Phrase (Private)",
						Value:  fmt.Sprintf("||`%s`||", mnemonic),
						Inline: false,
					},
					{
						Name:   "Time to Find",
						Value:  duration.Round(time.Second).String(),
						Inline: true,
					},
					{
						Name:   "Timestamp",
						Value:  time.Now().Format(time.RFC1123),
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
