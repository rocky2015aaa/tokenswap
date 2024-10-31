package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/asaskevich/govalidator"
	"golang.org/x/term"
)

const (
	//confirmationSentence = "I confirm, tokenswap cannot recover my password!"
	ConfirmationSentence = "test"
)

func InputUserEmail() (string, error) {
	fmt.Print("Enter your email address: ")
	reader := bufio.NewReader(os.Stdin)
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading email: %w", err)
	}
	// Validate email format (using a library or regex)
	if !govalidator.IsEmail(strings.TrimSpace(email)) {
		return "", fmt.Errorf("invalid email format. Please try again")
	}
	// Trim any leading or trailing whitespace
	return strings.TrimSpace(email), nil
}

func GetPassword() (string, error) {
	password, err := InputPassword("Enter new password: ")
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}

	confirmPassword, err := InputPassword("Confirm new password: ")
	if err != nil {
		return "", fmt.Errorf("error reading confirm password: %w", err)
	}

	if password != confirmPassword {
		fmt.Println("Passwords do not match. Please try again.")
		return "", fmt.Errorf("passwords do not match")
	}

	return password, nil
}

func InputPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}
	fmt.Println() // Print a newline since ReadPassword doesn't
	return strings.TrimSpace(string(bytePassword)), nil
}

func ConfirmPasswordWarning(confirmationSentence string) (bool, error) {
	fmt.Println("Losing your password means losing your account. We can't recover your password!")
	fmt.Printf("To confirm, please write: %s \n", confirmationSentence)
	// Read the entire line of input
	reader := bufio.NewReader(os.Stdin)
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("an error occurred while reading confirmation: %w", err)
	}
	// Trim any leading or trailing whitespace
	confirmation = strings.TrimSpace(confirmation)
	if confirmation != confirmationSentence {
		return false, fmt.Errorf("confirmation failed. Please try again")
	}
	return true, nil
}
