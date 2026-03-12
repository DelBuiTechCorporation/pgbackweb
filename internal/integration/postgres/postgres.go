package postgres

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/orsinium-labs/enum"
)

/*
	Important:
	Versions supported by PG Back Web must be supported in PostgreSQL Version Policy
	https://www.postgresql.org/support/versioning/

	Backing up a database from an old unsupported version should not be allowed.
*/

type version struct {
	Version string
	PGDump  string
	PSQL    string
}

type PGVersion enum.Member[version]

var (
	PG13 = PGVersion{version{
		Version: "13",
		PGDump:  "/usr/lib/postgresql/13/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/13/bin/psql",
	}}
	PG14 = PGVersion{version{
		Version: "14",
		PGDump:  "/usr/lib/postgresql/14/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/14/bin/psql",
	}}
	PG15 = PGVersion{version{
		Version: "15",
		PGDump:  "/usr/lib/postgresql/15/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/15/bin/psql",
	}}
	PG16 = PGVersion{version{
		Version: "16",
		PGDump:  "/usr/lib/postgresql/16/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/16/bin/psql",
	}}
	PG17 = PGVersion{version{
		Version: "17",
		PGDump:  "/usr/lib/postgresql/17/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/17/bin/psql",
	}}
	PG18 = PGVersion{version{
		Version: "18",
		PGDump:  "/usr/lib/postgresql/18/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/18/bin/psql",
	}}

	PGVersions     = []PGVersion{PG13, PG14, PG15, PG16, PG17, PG18}
	PGVersionsDesc = []PGVersion{PG18, PG17, PG16, PG15, PG14, PG13}
)

type Client struct{}

func New() *Client {
	return &Client{}
}

// ParseVersion returns the PGVersion enum member for the given PostgreSQL
// version as a string.
func (Client) ParseVersion(version string) (PGVersion, error) {
	switch version {
	case "13":
		return PG13, nil
	case "14":
		return PG14, nil
	case "15":
		return PG15, nil
	case "16":
		return PG16, nil
	case "17":
		return PG17, nil
	case "18":
		return PG18, nil
	default:
		return PGVersion{}, fmt.Errorf("pg version not allowed: %s", version)
	}
}

// Test tests the connection to the PostgreSQL database
func (Client) Test(version PGVersion, connString string) error {
	cmd := exec.Command(version.Value.PSQL, connString, "-c", "SELECT 1;")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql test v%s: %s",
			version.Value.Version, output,
		)
	}

	return nil
}

// DumpParams contains the parameters for the pg_dump command
type DumpParams struct {
	// DataOnly (--data-only): Dump only the data, not the schema (data definitions).
	// Table data, large objects, and sequence values are dumped.
	DataOnly bool

	// SchemaOnly (--schema-only): Dump only the object definitions (schema), not data.
	SchemaOnly bool

	// Clean (--clean): Output commands to DROP all the dumped database objects
	// prior to outputting the commands for creating them. This option is useful
	// when the restore is to overwrite an existing database. If any of the
	// objects do not exist in the destination database, ignorable error messages
	// will be reported during restore, unless --if-exists is also specified.
	Clean bool

	// IfExists (--if-exists): Use DROP ... IF EXISTS commands to drop objects in
	// --clean mode. This suppresses “does not exist” errors that might otherwise
	// be reported. This option is not valid unless --clean is also specified.
	IfExists bool

	// Create (--create): Begin the output with a command to create the database
	// itself and reconnect to the created database. (With a script of this form,
	// it doesn't matter which database in the destination installation you
	// connect to before running the script.) If --clean is also specified, the
	// script drops and recreates the target database before reconnecting to it.
	Create bool

	// NoComments (--no-comments): Do not dump comments.
	NoComments bool
}

// Dump runs the pg_dump command with the given parameters. It returns the SQL
// dump as an io.Reader.
func (Client) Dump(
	version PGVersion, connString string, params ...DumpParams,
) io.Reader {
	pickedParams := DumpParams{}
	if len(params) > 0 {
		pickedParams = params[0]
	}

	args := []string{connString}
	if pickedParams.DataOnly {
		args = append(args, "--data-only")
	}
	if pickedParams.SchemaOnly {
		args = append(args, "--schema-only")
	}
	if pickedParams.Clean {
		args = append(args, "--clean")
	}
	if pickedParams.IfExists {
		args = append(args, "--if-exists")
	}
	if pickedParams.Create {
		args = append(args, "--create")
	}
	if pickedParams.NoComments {
		args = append(args, "--no-comments")
	}

	errorBuffer := &bytes.Buffer{}
	reader, writer := io.Pipe()
	cmd := exec.Command(version.Value.PGDump, args...)
	cmd.Stdout = writer
	cmd.Stderr = errorBuffer

	go func() {
		defer writer.Close()
		if err := cmd.Run(); err != nil {
			writer.CloseWithError(fmt.Errorf(
				"error running pg_dump v%s: %s",
				version.Value.Version, errorBuffer.String(),
			))
		}
	}()

	return reader
}

// DumpZip runs the pg_dump command with the given parameters and returns the
// ZIP-compressed SQL dump as an io.Reader. compressionLevel follows compress/flate
// levels (0=Store, 1=BestSpeed … 9=BestCompression). Use -1 for default level (6).
func (c *Client) DumpZip(
	version PGVersion, connString string, compressionLevel int, params ...DumpParams,
) io.Reader {
	dumpReader := c.Dump(version, connString, params...)
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		zipWriter := zip.NewWriter(writer)
		if compressionLevel != 0 {
			level := compressionLevel
			zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
				return flate.NewWriter(out, level)
			})
		}
		defer zipWriter.Close()

		method := zip.Deflate
		if compressionLevel == 0 {
			method = zip.Store
		}
		fileWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:   "dump.sql",
			Method: method,
		})
		if err != nil {
			writer.CloseWithError(fmt.Errorf("error creating zip file: %w", err))
			return
		}

		if _, err := io.Copy(fileWriter, dumpReader); err != nil {
			writer.CloseWithError(fmt.Errorf("error writing to zip file: %w", err))
			return
		}
	}()

	return reader
}

// ZipPart represents a single ZIP file part created by DumpZipParts.
type ZipPart struct {
	FilePath string
	Size     int64
}

// countingWriter wraps an io.Writer and counts the total bytes written to it.
type countingWriter struct {
	w     io.Writer
	count int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.count += int64(n)
	return n, err
}

// DumpZipParts runs pg_dump and splits the output into multiple ZIP files,
// each at most maxPartSize compressed bytes. compressionLevel follows compress/flate
// levels (0=Store, 1=BestSpeed … 9=BestCompression). Use -1 for default level (6).
// The parts are stored as temp files in a new temp directory. The caller MUST defer
// os.RemoveAll(tempDir) after all parts have been consumed.
//
// Returns the list of parts, the temp directory path, and any error.
func (c *Client) DumpZipParts(
	version PGVersion, connString string, maxPartSize int64, compressionLevel int, params ...DumpParams,
) ([]ZipPart, string, error) {
	dumpReader := c.Dump(version, connString, params...)

	workDir, err := os.MkdirTemp("", "pbw-parts-*")
	if err != nil {
		return nil, "", fmt.Errorf("error creating temp dir: %w", err)
	}

	const safetyMargin = 2 * 1024 * 1024 // 2MB to absorb deflate internal buffering
	buf := make([]byte, 64*1024)          // 64KB read buffer
	var parts []ZipPart
	partNum := 1
	dumpDone := false

	for !dumpDone {
		partPath := strutil.CreatePath(true, workDir, fmt.Sprintf("part-%03d.zip", partNum))
		partFile, err := os.Create(partPath)
		if err != nil {
			return parts, workDir, fmt.Errorf("error creating part file: %w", err)
		}

		cw := &countingWriter{w: partFile}
		zw := zip.NewWriter(cw)
		if compressionLevel != 0 {
			level := compressionLevel
			zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
				return flate.NewWriter(out, level)
			})
		}

		method := zip.Deflate
		if compressionLevel == 0 {
			method = zip.Store
		}
		fw, err := zw.CreateHeader(&zip.FileHeader{
			Name:   fmt.Sprintf("dump-%03d.sql", partNum),
			Method: method,
		})
		if err != nil {
			partFile.Close()
			return parts, workDir, fmt.Errorf("error creating zip entry: %w", err)
		}

		for cw.count < maxPartSize-safetyMargin {
			n, readErr := dumpReader.Read(buf)
			if n > 0 {
				if _, writeErr := fw.Write(buf[:n]); writeErr != nil {
					partFile.Close()
					return parts, workDir, fmt.Errorf("error writing to zip: %w", writeErr)
				}
			}
			if readErr == io.EOF {
				dumpDone = true
				break
			}
			if readErr != nil {
				partFile.Close()
				return parts, workDir, fmt.Errorf("error reading dump: %w", readErr)
			}
		}

		if err := zw.Close(); err != nil {
			partFile.Close()
			return parts, workDir, fmt.Errorf("error closing zip writer: %w", err)
		}
		partFile.Close()

		fi, err := os.Stat(partPath)
		if err != nil {
			return parts, workDir, fmt.Errorf("error stating part file: %w", err)
		}
		if fi.Size() == 0 {
			// Empty part: dump ended exactly on a part boundary
			os.Remove(partPath)
			break
		}

		parts = append(parts, ZipPart{FilePath: partPath, Size: fi.Size()})
		partNum++
	}

	return parts, workDir, nil
}

// extractFirstSQLFromZip extracts the first .sql file found in the ZIP to destPath.
func extractFirstSQLFromZip(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("error opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".sql") {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("error opening zip entry: %w", err)
			}
			defer rc.Close()

			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("error creating output file: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, rc); err != nil {
				return fmt.Errorf("error extracting zip entry: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no .sql file found in zip: %s", zipPath)
}

// RestoreZipParts downloads or copies multiple ZIP parts (each containing a SQL
// chunk), concatenates all chunks, and runs psql to restore the database.
// Supports both single-file (legacy) and multi-part backups.
func (Client) RestoreZipParts(
	version PGVersion, connString string, isLocal bool, zipURLsOrPaths []string,
) error {
	workDir, err := os.MkdirTemp("", "pbw-restore-*")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	var sqlFiles []string
	for i, urlOrPath := range zipURLsOrPaths {
		zipPath := strutil.CreatePath(true, workDir, fmt.Sprintf("part-%03d.zip", i+1))

		if isLocal {
			cmd := exec.Command("cp", urlOrPath, zipPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error copying part %d to temp dir: %s", i+1, output)
			}
		} else {
			cmd := exec.Command("wget", "--no-verbose", "-O", zipPath, urlOrPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error downloading part %d: %s", i+1, output)
			}
		}

		if _, statErr := os.Stat(zipPath); os.IsNotExist(statErr) {
			return fmt.Errorf("part %d zip file not found after download/copy", i+1)
		}

		sqlFile := strutil.CreatePath(true, workDir, fmt.Sprintf("chunk-%03d.sql", i+1))
		if err := extractFirstSQLFromZip(zipPath, sqlFile); err != nil {
			return fmt.Errorf("error extracting part %d: %w", i+1, err)
		}
		sqlFiles = append(sqlFiles, sqlFile)
	}

	// Concatenate all SQL chunks into a single dump file
	dumpPath := strutil.CreatePath(true, workDir, "dump.sql")
	catArgs := append([]string{}, sqlFiles...)
	catCmd := exec.Command("cat", catArgs...)
	dumpFile, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("error creating merged dump file: %w", err)
	}
	catCmd.Stdout = dumpFile
	if mergeErr := catCmd.Run(); mergeErr != nil {
		dumpFile.Close()
		return fmt.Errorf("error merging SQL chunks: %w", mergeErr)
	}
	dumpFile.Close()

	cmd := exec.Command(version.Value.PSQL, connString, "-f", dumpPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql v%s command: %s",
			version.Value.Version, output,
		)
	}
	return nil
}

// RestoreZip downloads or copies the ZIP from the given url or path, unzips it,
// and runs the psql command to restore the database.
//
// The ZIP file must contain a dump.sql file with the SQL dump to restore.
//
//   - version: PostgreSQL version to use for the restore
//   - connString: connection string to the database
//   - isLocal: whether the ZIP file is local or a URL
//   - zipURLOrPath: URL or path to the ZIP file
func (Client) RestoreZip(
	version PGVersion, connString string, isLocal bool, zipURLOrPath string,
) error {
	workDir, err := os.MkdirTemp("", "pbw-restore-*")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	zipPath := strutil.CreatePath(true, workDir, "dump.zip")
	dumpPath := strutil.CreatePath(true, workDir, "dump.sql")

	if isLocal {
		cmd := exec.Command("cp", zipURLOrPath, zipPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error copying ZIP file to temp dir: %s", output)
		}
	}

	if !isLocal {
		cmd := exec.Command("wget", "--no-verbose", "-O", zipPath, zipURLOrPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error downloading ZIP file: %s", output)
		}
	}

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return fmt.Errorf("zip file not found: %s", zipPath)
	}

	cmd := exec.Command("unzip", "-o", zipPath, "dump.sql", "-d", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error unzipping ZIP file: %s", output)
	}

	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("dump.sql file not found in ZIP file: %s", zipPath)
	}

	cmd = exec.Command(version.Value.PSQL, connString, "-f", dumpPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql v%s command: %s",
			version.Value.Version, output,
		)
	}

	return nil
}
