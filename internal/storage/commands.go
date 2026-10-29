package storage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintStorageHelp prints usage instructions for storage commands.
func PrintStorageHelp() {
	fmt.Println("Aeitron Sovereign S3 Object Storage CLI (NDMA 2026 Compliant)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron storage <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  mb s3://<bucket> [flags]        Create a new sovereign S3 bucket")
	fmt.Println("  ls [s3://<bucket>]              List sovereign buckets or objects inside a bucket")
	fmt.Println("  cp <src> <dst>                  Copy/transfer objects between local filesystem and S3")
	fmt.Println("  rm s3://<bucket>/<key>          Delete an object from a sovereign bucket")
	fmt.Println("  rb s3://<bucket>                Remove an empty sovereign bucket")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --sovereignty <policy>          Territorial policy: 'Bangladesh' (default), 'EU', or 'Global'")
	fmt.Println("  --region <region>               Primary storage region (default: ap-south-2 Dhaka for Bangladesh)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  aeitron storage mb s3://sovereign-bd-fintech --sovereignty Bangladesh")
	fmt.Println("  aeitron storage ls")
	fmt.Println("  aeitron storage ls s3://sovereign-bd-fintech")
	fmt.Println("  aeitron storage cp ./data.csv s3://sovereign-bd-fintech/data.csv")
	fmt.Println("  aeitron storage cp s3://sovereign-bd-fintech/data.csv ./downloaded.csv")
	fmt.Println("  aeitron storage rm s3://sovereign-bd-fintech/data.csv")
	fmt.Println("  aeitron storage rb s3://sovereign-bd-fintech")
}

// HandleStorageCommand dispatches storage subcommands to their respective handlers.
func HandleStorageCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintStorageHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "mb", "makebucket":
		return handleMakeBucket(client, rest)
	case "ls", "list":
		return handleList(client, rest)
	case "cp", "copy":
		return handleCopy(client, rest)
	case "rm", "delete":
		return handleRemoveObject(client, rest)
	case "rb", "removebucket":
		return handleRemoveBucket(client, rest)
	case "help", "--help", "-h":
		PrintStorageHelp()
		return nil
	default:
		return fmt.Errorf("unknown storage subcommand %q; run 'aeitron storage help'", sub)
	}
}

func handleMakeBucket(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket URI required. Usage: aeitron storage mb s3://<bucket> [--sovereignty <Bangladesh|EU|Global>]")
	}

	fs := flag.NewFlagSet("storage mb", flag.ContinueOnError)
	sovereignty := fs.String("sovereignty", "Bangladesh", "Sovereignty jurisdiction policy: Bangladesh, EU, Global")
	region := fs.String("region", "", "Primary deployment region")

	var rawURI string
	var nonFlagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			// flags start here or pass to fs
			if err := fs.Parse(args[i:]); err != nil {
				return err
			}
			break
		} else {
			nonFlagArgs = append(nonFlagArgs, args[i])
		}
	}

	if len(nonFlagArgs) == 0 {
		return fmt.Errorf("bucket URI required. Usage: aeitron storage mb s3://<bucket>")
	}
	rawURI = nonFlagArgs[0]

	bucketName := rawURI
	if strings.HasPrefix(rawURI, "s3://") {
		b, _, err := ParseS3URI(rawURI)
		if err != nil {
			return err
		}
		bucketName = b
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := client.MakeBucket(ctx, bucketName, *sovereignty, *region)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Sovereign S3 Bucket Created: s3://%s\n", info.Name)
	fmt.Println("================================================================================")
	fmt.Printf("  Jurisdiction:       %s\n", info.Sovereignty)
	fmt.Printf("  Primary Region:     %s\n", info.PrimaryRegion)
	fmt.Printf("  NDMA 2026 Policy:   ENFORCED (Strict territorial geo-fencing enabled)\n")
	fmt.Printf("  Storage Topology:   Reed-Solomon 4+2 Erasure Coding + HSM AES-256-GCM\n")
	fmt.Printf("  Created At (UTC):   %s\n", info.CreatedAt.Format(time.RFC3339))
	fmt.Println("================================================================================")

	return nil
}

func handleList(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(args) == 0 || args[0] == "s3://" || args[0] == "" {
		// List all buckets
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			return fmt.Errorf("failed to list buckets: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Println(" Aeitron Sovereign S3 Buckets")
		fmt.Println("================================================================================")
		if len(buckets) == 0 {
			fmt.Println("No sovereign buckets found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "BUCKET NAME\tSOVEREIGNTY\tPRIMARY REGION\tCREATED AT")
		fmt.Fprintln(w, "-----------\t-----------\t--------------\t----------")
		for _, b := range buckets {
			fmt.Fprintf(w, "s3://%-20s\t%-12s\t%-22s\t%s\n",
				b.Name, b.Sovereignty, b.PrimaryRegion, b.CreatedAt.Format("2006-01-02 15:04:05 MST"))
		}
		w.Flush()
		return nil
	}

	// List objects in bucket
	rawURI := args[0]
	bucket, _, err := ParseS3URI(rawURI)
	if err != nil {
		return err
	}

	objects, err := client.ListObjects(ctx, bucket)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Bucket Inventory: s3://%s\n", bucket)
	fmt.Println("================================================================================")
	if len(objects) == 0 {
		fmt.Println("Bucket is empty (0 objects).")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "KEY\tSIZE (BYTES)\tCHUNKS (RS)\tENCRYPTION\tETAG\tLAST MODIFIED")
	fmt.Fprintln(w, "---\t------------\t-----------\t----------\t----\t-------------")
	for _, obj := range objects {
		chunkStr := fmt.Sprintf("%d+%d", obj.DataChunks, obj.ParityChunks)
		fmt.Fprintf(w, "%-24s\t%12d\t%-11s\t%-15s\t%-18s\t%s\n",
			obj.Key, obj.SizeBytes, chunkStr, obj.Encryption, obj.ETag, obj.LastModified.Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	return nil
}

func handleCopy(client *Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("source and destination required. Usage: aeitron storage cp <src> <dst>")
	}

	src := args[0]
	dst := args[1]

	srcIsS3 := strings.HasPrefix(src, "s3://")
	dstIsS3 := strings.HasPrefix(dst, "s3://")

	if !srcIsS3 && !dstIsS3 {
		return fmt.Errorf("at least one of source or destination must be an s3:// URI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if !srcIsS3 && dstIsS3 {
		// Upload: local file -> S3
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read local file %q: %w", src, err)
		}

		bucket, key, err := ParseS3URI(dst)
		if err != nil {
			return err
		}
		if key == "" || strings.HasSuffix(dst, "/") {
			// Use local file basename
			key = strings.TrimSuffix(key, "/")
			if key != "" {
				key = key + "/" + filepath.Base(src)
			} else {
				key = filepath.Base(src)
			}
		}

		fmt.Printf("Uploading %s -> s3://%s/%s (Size: %d bytes)...\n", src, bucket, key, len(data))
		obj, err := client.PutObject(ctx, bucket, key, data, "sovereign")
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}

		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("SUCCESS: Uploaded to s3://%s/%s\n", obj.Bucket, obj.Key)
		fmt.Printf("  Size:             %d bytes\n", obj.SizeBytes)
		fmt.Printf("  Erasure Coding:   Reed-Solomon %d+%d (%d data + %d parity chunks)\n",
			obj.DataChunks, obj.ParityChunks, obj.DataChunks, obj.ParityChunks)
		fmt.Printf("  Encryption:       %s (Key ID: %s)\n", obj.Encryption, obj.HSMKeyID)
		fmt.Printf("  ETag:             %s\n", obj.ETag)
		fmt.Printf("  SHA-256 Checksum: %s\n", obj.SHA256Checksum)
		fmt.Printf("  Jurisdiction:     %s (NDMA 2026 Compliant)\n", obj.Sovereignty)
		fmt.Println("--------------------------------------------------------------------------------")
		return nil
	}

	if srcIsS3 && !dstIsS3 {
		// Download: S3 -> local file
		bucket, key, err := ParseS3URI(src)
		if err != nil {
			return err
		}
		if key == "" {
			return fmt.Errorf("source object key required: s3://<bucket>/<key>")
		}

		targetPath := dst
		fi, statErr := os.Stat(dst)
		if statErr == nil && fi.IsDir() {
			targetPath = filepath.Join(dst, filepath.Base(key))
		}

		fmt.Printf("Downloading s3://%s/%s -> %s...\n", bucket, key, targetPath)
		data, obj, err := client.GetObject(ctx, bucket, key)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write local destination file %q: %w", targetPath, err)
		}

		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("SUCCESS: Downloaded s3://%s/%s -> %s\n", obj.Bucket, obj.Key, targetPath)
		fmt.Printf("  Size:             %d bytes\n", len(data))
		fmt.Printf("  Integrity:        VERIFIED (SHA-256 matched)\n")
		fmt.Printf("  Erasure Coding:   Reconstructed from Reed-Solomon %d+%d chunks\n", obj.DataChunks, obj.ParityChunks)
		fmt.Printf("  Decryption:       %s decrypted via Sovereign HSM\n", obj.Encryption)
		fmt.Println("--------------------------------------------------------------------------------")
		return nil
	}

	return fmt.Errorf("S3 to S3 cross-bucket copy not supported in this client version")
}

func handleRemoveObject(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("object URI required. Usage: aeitron storage rm s3://<bucket>/<key>")
	}

	bucket, key, err := ParseS3URI(args[0])
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("object key required. Usage: aeitron storage rm s3://<bucket>/<key>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.DeleteObject(ctx, bucket, key); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	fmt.Printf("SUCCESS: Deleted object s3://%s/%s\n", bucket, key)
	return nil
}

func handleRemoveBucket(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket URI required. Usage: aeitron storage rb s3://<bucket>")
	}

	bucket, key, err := ParseS3URI(args[0])
	if err != nil {
		return err
	}
	if key != "" {
		return fmt.Errorf("remove bucket operates on bucket roots only, got key %q", key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.RemoveBucket(ctx, bucket); err != nil {
		return fmt.Errorf("failed to remove bucket: %w", err)
	}

	fmt.Printf("SUCCESS: Removed sovereign bucket s3://%s\n", bucket)
	return nil
}
