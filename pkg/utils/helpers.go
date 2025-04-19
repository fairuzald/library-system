package utils

import (
	"math/rand"
	"strings"
	"time"

	"strconv"

	"github.com/google/uuid"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomString generates a random string of the given length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateUUID generates a new UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// NormalizeString normalizes a string for searching
func NormalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// SliceContains checks if a slice contains an element
func SliceContains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// Contains checks if a slice contains an element
func Contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Unique returns a new slice with duplicates removed
func Unique[T comparable](slice []T) []T {
	keys := make(map[T]bool)
	uniqueSlice := []T{}
	for _, item := range slice {
		if _, value := keys[item]; !value {
			keys[item] = true
			uniqueSlice = append(uniqueSlice, item)
		}
	}
	return uniqueSlice
}

// ParseInt parses a string to an integer with error handling
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ParseBool parses a string to a boolean with error handling
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

// StringPtr returns a pointer to the string value
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the int value
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the bool value
func BoolPtr(b bool) *bool {
	return &b
}
