package cli

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"

	"github.com/spf13/cobra"
)

var unsealCmd = &cobra.Command{
	Use:   "unseal [directory]",
	Short: "Decrypt a directory",
	Long:  `Unseal (decrypt) all files in a directory using the correct password.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) { // The function executed when 'aegis unseal' is run.
		dir := args[0] //Retrieves the directory path provided as the first argument.

		fmt.Printf("🔑 Attempting to unseal files in directory '%s'...\n", dir)
		fmt.Print("Enter password: ")

		// --- PASSWORD ERROR HANDLING ---
		pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd())) // Reads password from STDIN without showing input.
		if err != nil {                                        // Checks if reading the password failed.
			// Prints error to standard error stream (os.Stderr) and exits cleanly
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			return // Exit Run function immediately
		}
		password := string(pwdBytes)
		fmt.Println()
		// ---------------------------------------

		var filesUnsealed int // Counter for successfully unsealed files.
		var filesFailed int   // Counter for files that failed to unseal.
		var filesSkipped int  // Counter for files that were skipped.
		// walkErr captures any fatal error from the directory walk.

		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { // Starts recursively walking the directory.
			// If the walk encounters an error (like non-existent directory),
			// we must return the error itself to the main walkErr variable and trigger the Fatal Error block at the end.
			if err != nil {
				return err // Returns error to walkErr to trigger the Fatal Error block at the end.
			}
			if info.IsDir() { // Skips directories, only processing files.
				return nil
			}
			// Skip files that do not have the .aegis extension
			if !strings.HasSuffix(path, ".aegis") {
				filesSkipped++
				return nil
			}

			data, err := os.ReadFile(path) // Reads the entire sealed file into memory.
			if err != nil {                // Checks if reading the file failed.
				fmt.Printf("❌ Could not read sealed file %s: %v. Skipping.\n", path, err) // Prints error message for the specific file.
				filesFailed++                                                             // Increments failed counter.
				return nil                                                                // Skip to the next file
			}

			// Decryption Setup
			if len(data) < 16+12 { // Minimum length: 16 bytes salt + 12 bytes nonce(GCM is usually 12B).
				fmt.Printf("❌ Sealed file %s is too short/corrupted. Skipping.\n", path) // Prints error for malformed file.
				filesFailed++                                                            // Increments failed counter.
				return nil                                                               // Skip to the next file
			}

			salt := data[:16] // Extracts the salt from the start of the file.(used for key derivation).
			//Re-derive the key using Scrypt with the stored salt and the user's password.
			key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32) // Derives the encryption key using Scrypt.
			if err != nil {                                                 // Checks if key derivation failed.
				fmt.Printf("❌ Failed to derive key for %s: %v. Skipping.\n", path, err) // Prints error message.
				filesFailed++                                                           // Increments failed counter.
				return nil                                                              // Skip to the next file
			}
			//initialize AES-GCM for decryption
			block, err := aes.NewCipher(key) // Creates a new AES cipher block with the derived key.
			if err != nil {                  // Checks if cipher creation failed.
				filesFailed++                                                            // Increments failed counter.
				return fmt.Errorf("failed to create cipher block for %s: %v", path, err) // Returns error to be handled by walkErr.
			}

			gcm, err := cipher.NewGCM(block) // Sets up Galois/Counter Mode (GCM) for authenticated decryption.
			if err != nil {                  // Checks if GCM setup failed.
				filesFailed++                                                   // Increments failed counter.
				return fmt.Errorf("failed to create GCM for %s: %v", path, err) // Returns error to be handled by walkErr.
			}
			//seperate nonce and ciphertext
			nonceSize := gcm.NonceSize() // Retrieves the nonce size required by GCM.

			if len(data) < 16+nonceSize { // Validates that the file is long enough to contain salt + nonce + ciphertext.
				fmt.Printf("❌ Sealed file %s is malformed (missing nonce/ciphertext). Skipping.\n", path) // Prints error for malformed file.
				filesFailed++                                                                             //	 Increments failed counter.
				return nil                                                                                // Skip to the next file
			}

			nonce := data[16 : 16+nonceSize]  // Extracts the nonce from the file data.
			ciphertext := data[16+nonceSize:] //

			plaintextWithExt, err := gcm.Open(nil, nonce, ciphertext, nil) // Decrypts the ciphertext using AES-GCM.
			if err != nil {                                                // Checks if decryption failed (likely due to wrong password or corruption).
				fmt.Printf("⛔ Decryption FAILED for '%s': Wrong password or file corrupted.\n", filepath.Base(path)) // Prints decryption failure message.
				filesFailed++                                                                                        // Increments failed counter.
				return nil                                                                                           // Skip to the next file
			}

			// --- FILENAMELOGIC: Recover Extension ---
			nullIndex := -1                      // Index of the null terminator separating extension and content.
			for i, b := range plaintextWithExt { // Scans for the null terminator byte (0x00).
				if b == 0x00 { // Found the null terminator
					nullIndex = i // Sets the index and
					break         // exits the loop
				}
			}

			if nullIndex == -1 { // Null terminator not found
				fmt.Printf("Warning: Could not find original extension in '%s'. Assuming old format or corruption.\n", filepath.Base(path)) // Prints warning message.
				out := strings.TrimSuffix(path, ".aegis")                                                                                   // Constructs output filename by removing .aegis extension.
				os.WriteFile(out, plaintextWithExt, 0600)                                                                                   // Writes the decrypted data as-is (no extension).
				os.Remove(path)                                                                                                             // Deletes the original sealed file.
				filesFailed++                                                                                                               //	Increments failed counter.
				fmt.Println("Unsealed (Warning):", out)                                                                                     // Prints success message with warning.
				return nil                                                                                                                  // Skip to the next file
			}

			originalExt := string(plaintextWithExt[:nullIndex]) // Extracts the original file extension.
			plaintext := plaintextWithExt[nullIndex+1:]         // Extracts the actual plaintext data.
			// Construct output filename by appending the original extension

			base := strings.TrimSuffix(path, ".aegis") // Base filename without .aegis extension
			out := base + originalExt                  // Joins base with the recovered original extension

			if err := os.WriteFile(out, plaintext, 0600); err != nil { // Writes the decrypted plaintext to the new file.
				fmt.Printf("❌ Failed to write unsealed file %s: %v. Skipping.\n", out, err) // Prints error message.
				filesFailed++                                                               // Increments failed counter.
				return nil                                                                  // Skip to the next file
			}

			if err := os.Remove(path); err != nil { // Deletes the original sealed file.
				fmt.Printf("Warning: Failed to remove sealed file %s: %v\n", path, err) // Warns if deletion fails.
			}

			filesUnsealed++                                                   // Increments success counter.
			fmt.Printf("✅ Unsealed '%s' -> '%s'\n", filepath.Base(path), out) // Prints success message.
			return nil                                                        // Continues to the next file
		})

		if walkErr != nil { // Checks if a fatal error occurred during the directory walk.
			fmt.Printf("\n\n🔥 Fatal Error during unsealing: %v\n", walkErr) // Prints the fatal error message.
			os.Exit(1)                                                      // Exits the program with a non-zero status code.
		}

		// Final summary output
		fmt.Printf("\n✨ Unsealing complete for directory '%s'.\n", dir)   // Prints completion message.
		fmt.Printf("   Successfully unsealed %d files.\n", filesUnsealed) // Prints count of successfully unsealed files.
		if filesFailed > 0 {                                              // Prints failed count only if necessary.
			fmt.Printf("   Failed to unseal %d files (wrong password, corruption, or old format).\n", filesFailed) // Prints count of failed files.
		}
		if filesSkipped > 0 { // Prints skipped count only if necessary.
			fmt.Printf("   Skipped %d files (did not have '.aegis' extension).\n", filesSkipped) // Prints count of skipped files.
		}
	},
}

func init() {
	RootCmd.AddCommand(unsealCmd)
}
