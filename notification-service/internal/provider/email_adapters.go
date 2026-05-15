package provider

import (
	"errors"
	"log"
	"math/rand"
	"time"
)

type MockEmailSender struct{}

func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{}
}

func (m *MockEmailSender) SendEmail(to string, subject string, body string) error {
	// Simulate network latency
	time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

	// Simulate occasional random failures (20% chance)
	if rand.Float32() < 0.2 {
		return errors.New("temporary external provider failure")
	}

	log.Printf("[MOCK EMAIL] Sent to %s: %s", to, subject)
	return nil
}

type RealEmailSender struct{}

func NewRealEmailSender() *RealEmailSender {
	return &RealEmailSender{}
}

func (r *RealEmailSender) SendEmail(to string, subject string, body string) error {
	// In a real scenario, this would use Mailjet/SMTP
	// For this assignment, we simulate "Real" conditions
	log.Printf("[REAL EMAIL] Sending via external API to %s...", to)
	time.Sleep(800 * time.Millisecond)
	log.Printf("[REAL EMAIL] Successfully delivered to %s", to)
	return nil
}
