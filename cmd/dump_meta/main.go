package main
import (
  "context"
  "encoding/json"
  "flag"
  "fmt"
  "log"
  "os"
  "time"
  chroma "github.com/hetulpatel/Arbitrage/internal/chroma"
)
func main(){
  id := flag.String("id", "", "market id")
  flag.Parse()
  if *id == "" { log.Fatal("-id required") }
  chromaURL := os.Getenv("CHROMA_URL")
  if chromaURL == "" { chromaURL = "http://chromadb:8000" }
  collection := os.Getenv("CHROMA_COLLECTION")
  if collection == "" { collection = "market_snapshots" }
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  client := chroma.NewClient(chromaURL)
  col, err := client.GetCollection(ctx, collection)
  if err != nil { log.Fatal(err) }
  resp, err := client.Get(ctx, col.ID, chroma.GetRequest{Where: map[string]any{"market_id": *id}, Include: []string{"metadatas"}})
  if err != nil { log.Fatal(err) }
  b, _ := json.MarshalIndent(resp.Metadatas, "", "  ")
  fmt.Println(string(b))
}
