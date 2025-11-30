package cli

import (
	"crypto/aes"    // Standard library for AES encryption.
	"crypto/cipher" // Standard library for cipher modes (GCM).
	"crypto/rand"   // Source for cryptographically secure random numbers (salt, nonce).
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/scrypt" // Industry-standard package for Key Derivation Function (KDF).
	"golang.org/x/term"          // Used to read password without echoing to the console.

	"github.com/spf13/cobra"
)

// var sealCmd defines the structure and metadata for the 'aegis seal' command.
var sealCmd = &cobra.Command{
	Use:   "seal [directory]",                                                              // Defines the command usage syntax.
	Short: "Encrypt a directory",                                                           // A brief, one-line summary of the command.
	Long:  `Seal (encrypt) a directory and all its contents using a password-derived key.`, // A detailed description.
	Args:  cobra.MinimumNArgs(1),                                                           // Ensures at least one argument (the directory path) is provided.
	Run: func(cmd *cobra.Command, args []string) { // The function executed when 'aegis seal' is run.
		dir := args[0] // Retrieves the directory path provided as the first argument.

		fmt.Printf("🔒 Securing directory '%s'...\n", dir)
		fmt.Print("Enter password: ")

		// Reads password from STDIN without showing input.
		pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil { // Checks if reading the password failed.
			// Prints error to standard error stream (os.Stderr) and exits cleanly
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err) // Prints error to the standard error stream.
			return                                                      // Exit Run function immediately
		}
		password := string(pwdBytes) // Converts the secure byte slice password into a string.
		fmt.Println()                // Prints a newline character after password input.

		// Placeholder for exclusion logic
		excludeList := []string{".git", "vendor", "node_modules", "target"} // Default list of items to skip.
		excludeSet := make(map[string]bool)                                 // Creates a map for fast lookup of exclusions.
		for _, item := range excludeList {                                  // Iterates through the list.
			excludeSet[item] = true // Populates the map.
		}

		var filesSealed int  // Counter for successfully sealed files.
		var filesSkipped int // Counter for skipped files.
		// walkErr captures any fatal error from the directory walk.
		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			// If the walk encounters an error (like non-existent directory),
			// we must return the error itself to the main walkErr variable and trigger the Fatal Error block at the end.
			if err != nil {
				return err // Returns the error, stopping the walk and populating walkErr.
			}

			// Exclusion and Symlink checks (Filtering Logic)
			if info.IsDir() { // Checks if the current path is a directory.
				if excludeSet[info.Name()] { // Checks if the directory name is in the exclusion list.
					fmt.Printf("   Skipping excluded directory: %s\n", info.Name())
					return filepath.SkipDir // Skip this directory and its contents
				}
				if path == dir {
					return nil
				}
				return nil // Continues traversal into subdirectories.
			}

			if (info.Mode() & os.ModeSymlink) != 0 { // Checks if the file is a symbolic link.
				fmt.Printf("   Skipping symbolic link: %s\n", path)
				filesSkipped++
				return nil // Skips symlinks for security/robustness.
			}

			if strings.HasSuffix(path, ".aegis") { // Checks if the file is already sealed.
				filesSkipped++
				return nil // Skips already sealed files.
			}

			plaintext, err := os.ReadFile(path) // Reads the entire file content into memory.
			if err != nil {                     // Checks for file read errors (e.g., permissions).
				fmt.Printf("❌ Could not read file %s: %v. Skipping.\n", path, err)
				return nil // Skip this file, but continue the walk
			}

			// Crypto Setup
			// 1. Salt Generation: Unique, 16-byte random salt for every file.
			salt := make([]byte, 16)                   // Creates a 16-byte buffer for the unique salt.
			if _, err := rand.Read(salt); err != nil { // Fills the salt buffer with CSPRNG data.
				return fmt.Errorf("failed to generate salt for %s: %v", path, err) // Returns error for fatal crypto failure.
			}
			// 2. Key Derivation: Scrypt generates a strong 32-byte key (AES-256) from the password + salt.
			key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
			if err != nil {
				return fmt.Errorf("failed to derive key for %s: %v", path, err)
			}
			// 3. GCM Setup: Initializes AES in Galois/Counter Mode (GCM) for authenticated encryption.
			block, err := aes.NewCipher(key) // Creates the AES block cipher instance.
			if err != nil {
				return fmt.Errorf("failed to create cipher block for %s: %v", path, err)
			}

			gcm, err := cipher.NewGCM(block) // Sets up Galois/Counter Mode (GCM) for authenticated encryption.
			if err != nil {
				return fmt.Errorf("failed to create GCM for %s: %v", path, err)
			}
			// 4. Nonce Generation: Unique, random Initialization Vector (IV) for the encryption.
			nonce := make([]byte, gcm.NonceSize())                     // Creates a buffer for the Initialization Vector (Nonce)
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil { // Fills the nonce buffer with random data.
				return fmt.Errorf("failed to generate nonce for %s: %v", path, err)
			}

			// --- FILENAME LOGIC: Embed Extension ---
			// Embed the original file extension (e.g., .txt) into the encrypted data.
			originalExt := []byte(filepath.Ext(path))                 // Extracts the original extension (e.g., .txt).
			plaintextWithExt := append(originalExt, 0x00)             // Null terminator separates extension
			plaintextWithExt = append(plaintextWithExt, plaintext...) // Appends the actual file content to be encrypted.

			// 5. Encryption: Seals the data using GCM (output includes ciphertext and authentication tag).
			ciphertext := gcm.Seal(nil, nonce, plaintextWithExt, nil)
			// Final file format: [Salt] + [Nonce] + [Ciphertext + Auth Tag]
			final := append(salt, append(nonce, ciphertext...)...)

			// Construct the clean output filename (remove original extension, add .aegis)
			baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) // Removes old extension from filename.
			dirPath := filepath.Dir(path)                                           // Gets the directory part of the path.
			out := filepath.Join(dirPath, baseName+".aegis")                        // Joins path with the new masked filename.
			// Write output and clean up original file.
			if err := os.WriteFile(out, final, 0600); err != nil { // Writes the final encrypted data to the new file.
				return fmt.Errorf("failed to write sealed file %s: %v", out, err)
			}

			if err := os.Remove(path); err != nil { // Deletes the original plaintext file.
				fmt.Printf("Warning: Failed to remove original file %s: %v\n", path, err) // Warns if deletion fails.
			}

			filesSealed++                                                   // Increments success counter.
			fmt.Printf("✅ Sealed '%s' -> '%s'\n", path, filepath.Base(out)) //Prints success message.
			return nil                                                      // Returns nil to continue the filepath.Walk traversal.
		})
		// Check for fatal errors from filepath.Walk
		if walkErr != nil { // Checks if the CRITICAL FIX triggered (i.e., a fatal error occurred).
			// This block catches the fatal error returned from filepath.Walk (e.g., non-existent directory)
			fmt.Printf("\n\n🔥 Fatal Error during sealing: %v\n", walkErr)
			os.Exit(1) // Exits the program with a non-zero status code (failure).

		}

		// Final summary output
		fmt.Printf("\n✨ Sealing complete for directory '%s'.\n", dir)
		fmt.Printf("   Successfully sealed %d files.\n", filesSealed)
		if filesSkipped > 0 { // Prints skipped items only if necessary.
			fmt.Printf("   Skipped %d items (already sealed, symlinks, or excluded).\n", filesSkipped)
		}
	},
}

func init() {
	RootCmd.AddCommand(sealCmd)
}
