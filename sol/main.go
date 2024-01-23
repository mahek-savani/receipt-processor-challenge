package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// receipt structure
type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

// items structure
type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

// storing the calculated points for each receipt
var ReceiptPoints = make(map[string]int)

func main() {
	http.HandleFunc("/receipts/process", processReceipt)
	http.HandleFunc("/receipts/", getPoints)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func processReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Error decoding JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	points := totalPoints(receipt)
	receiptID := uuid.NewString()

	ReceiptPoints[receiptID] = points

	json.NewEncoder(w).Encode(map[string]string{"id": receiptID})
}

func totalPoints(receipt Receipt) int {
	points := 0

	//points for alphanumeric characters in retailer name
	for _, char := range receipt.Retailer {
		if isAlphaNumeric(char) {
			points++
		}
	}

	//50 points if total is a round dollar amount
	if validateDollar(receipt.Total) {
		points += 50
	}

	//25 points if total is a multiple of 0.25
	if validatePrice(receipt.Total) {
		points += 25
	}

	//5 points for every two items
	points += (len(receipt.Items) / 2) * 5

	for _, item := range receipt.Items {
		points += validateDesc(item)
	}

	//validation for purchase date
	date, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err != nil {
		log.Println("Error parsing date:", err)
	} else {
		if date.Day()%2 != 0 {
			points += 6
		}
	}

	//validation for purchase time
	time, err := time.Parse("15:04", receipt.PurchaseTime)
	if err != nil {
		log.Println("Error parsing time:", err)
	} else {
		if time.Hour() >= 14 && time.Hour() < 16 {
			points += 10
		}
	}

	return points
}

func isAlphaNumeric(r rune) bool {
	return strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r)
}

func validateDollar(total string) bool {
	if strings.HasSuffix(total, ".00") {
		return true
	}
	return false
}

func validatePrice(total string) bool {
	value, err := strconv.ParseFloat(total, 64)
	if err != nil {
		return false
	}
	//convert value to an integer
	intValue := int(value * 100)
	return intValue%25 == 0
}

func validateDesc(item Item) int {
	itemPoints := 0

	//yrim item description and check if its length is a multiple of 3
	trimmedDesc := strings.TrimSpace(item.ShortDescription)
	if len(trimmedDesc)%3 == 0 {
		price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			return 0
		}

		additionalPoints := math.Ceil(price * 0.2)
		itemPoints = int(additionalPoints)
	}
	return itemPoints
}

func getPoints(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	segments := strings.Split(path, "/")
	if len(segments) < 4 {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}
	if segments[3] != "points" {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}
	id := segments[2]
	// log.Println("ID", id, len(segments))

	points, exists := ReceiptPoints[id]
	if !exists {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"points": points})
}
