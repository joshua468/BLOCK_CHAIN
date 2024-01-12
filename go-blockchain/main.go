package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Block struct {
	Pos       int
	Data      Bookcheckout
	Hash      string
	PrevHash  string
	Timestamp string
}

type Bookcheckout struct {
	BookID       string `json:"bookid"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"`
}

type Blockchain struct {
	blocks []*Block
}

var BlockChain *Blockchain

func (b *Block) generateHash() {
	bytes, _ := json.Marshal(b.Data)
	data := string(b.Pos) + b.Timestamp + string(bytes) + b.PrevHash
	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevblock *Block, checkoutitem Bookcheckout) *Block {
	block := &Block{}
	block.Pos = prevblock.Pos + 1

	block.Timestamp = time.Now().String()
	block.PrevHash = prevblock.Hash
	block.generateHash()
	return block
}

func (bc *Blockchain) AddBlock(data Bookcheckout) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	block := CreateBlock(prevBlock, data)

	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}
}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}
	if !block.validateHash(block.Hash) {
		return false
	}
	if prevBlock.Pos+1 != block.Pos {
		return false
	}
	return true
}

func (b *Block) validateHash(hash string) bool {
	b.generateHash()
	return b.Hash == hash
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkoutitem Bookcheckout

	if err := json.NewDecoder(r.Body).Decode(&checkoutitem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not write block: %v", err)
		w.Write([]byte(fmt.Sprintf("could not write block: %v", err)))
		return
	}

	BlockChain.AddBlock(checkoutitem)
}

func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not create: %v", err)
		w.Write([]byte("could not create new book"))
		return
	}
	h := md5.New()
	io.WriteString(h, book.ISBN+book.PublishDate)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))

	resp, err := json.MarshalIndent(book, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not save book data"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, Bookcheckout{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{blocks: []*Block{GenesisBlock()}}
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	jbytes, err := json.MarshalIndent(BlockChain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	io.WriteString(w, string(jbytes))
}

func main() {
	BlockChain = NewBlockchain()
	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newBook).Methods("POST")

	go func() {
		for _, block := range BlockChain.blocks {
			fmt.Printf("Prev.Hash:%x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data:%v\n", string(bytes))
			fmt.Printf("Hash:%v\n", block.Hash)
			fmt.Println()
		}
	}()

	log.Println("listening at port 3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}
