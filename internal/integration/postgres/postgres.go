package postgres

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

	PGVersions = []PGVersion{PG13, PG14, PG15, PG16, PG17}
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

// ListDatabases lists all non-template databases in the PostgreSQL instance
func (Client) ListDatabases(version PGVersion, connString string) ([]string, error) {
	cmd := exec.Command(version.Value.PSQL, connString, "-At", "-c", "SELECT datname FROM pg_database WHERE datistemplate = false;")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"error running psql list databases v%s: %s",
			version.Value.Version, output,
		)
	}

	var databases []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			databases = append(databases, trimmed)
		}
	}

	return databases, nil
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
// ZIP-compressed SQL dump as an io.Reader.
// If allDatabases is true, it will dump all non-template databases into separate files in the ZIP.
// If zipPassword is provided, the ZIP will be password-protected.
func (c *Client) DumpZip(
	version PGVersion, connString string, allDatabases bool, zipPassword string, params ...DumpParams,
) io.Reader {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		// Create temp directory for SQL files
		tempDir, err := os.MkdirTemp("", "pbw-dump-*")
		if err != nil {
			writer.CloseWithError(fmt.Errorf("error creating temp dir: %w", err))
			return
		}
		defer os.RemoveAll(tempDir)

		if allDatabases {
			// Parse the connection string to get base URL without dbname
			u, err := url.Parse(connString)
			if err != nil {
				writer.CloseWithError(fmt.Errorf("error parsing connection string: %w", err))
				return
			}
			baseConnString := u.String()
			pathParts := strings.Split(u.Path, "/")
			if len(pathParts) > 1 {
				// Remove the dbname from path
				u.Path = strings.Join(pathParts[:len(pathParts)-1], "/")
				baseConnString = u.String()
			}

			// List all databases
			databases, err := c.ListDatabases(version, connString)
			if err != nil {
				writer.CloseWithError(fmt.Errorf("error listing databases: %w", err))
				return
			}

			// Dump each database to temp files
			sqlFiles := make([]string, 0, len(databases))
			for _, db := range databases {
				// Create connection string for this database
				dbConnString := baseConnString
				if strings.HasSuffix(baseConnString, "/") {
					dbConnString += db
				} else {
					dbConnString += "/" + db
				}

				dumpReader := c.Dump(version, dbConnString, params...)
				sqlPath := strutil.CreatePath(true, tempDir, db+".sql")
				sqlFiles = append(sqlFiles, sqlPath)

				sqlFile, err := os.Create(sqlPath)
				if err != nil {
					writer.CloseWithError(fmt.Errorf("error creating temp SQL file for %s: %w", db, err))
					return
				}

				if _, err := io.Copy(sqlFile, dumpReader); err != nil {
					sqlFile.Close()
					writer.CloseWithError(fmt.Errorf("error writing dump for %s: %w", db, err))
					return
				}
				sqlFile.Close()
			}

			// Create ZIP with all SQL files
			if err := c.createZipWithPassword(writer, tempDir, sqlFiles, zipPassword); err != nil {
				writer.CloseWithError(fmt.Errorf("error creating ZIP: %w", err))
				return
			}

		} else {
			// Single database dump
			dumpReader := c.Dump(version, connString, params...)
			sqlPath := strutil.CreatePath(true, tempDir, "dump.sql")

			sqlFile, err := os.Create(sqlPath)
			if err != nil {
				writer.CloseWithError(fmt.Errorf("error creating temp SQL file: %w", err))
				return
			}

			if _, err := io.Copy(sqlFile, dumpReader); err != nil {
				sqlFile.Close()
				writer.CloseWithError(fmt.Errorf("error writing dump: %w", err))
				return
			}
			sqlFile.Close()

			// Create ZIP with single SQL file
			sqlFiles := []string{sqlPath}
			if err := c.createZipWithPassword(writer, tempDir, sqlFiles, zipPassword); err != nil {
				writer.CloseWithError(fmt.Errorf("error creating ZIP: %w", err))
				return
			}
		}
	}()

	return reader
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

// createZipWithPassword creates a ZIP file with optional password protection using the system zip command
func (c *Client) createZipWithPassword(writer io.Writer, tempDir string, sqlFiles []string, zipPassword string) error {
	// Create a temporary ZIP file
	zipPath := strutil.CreatePath(true, tempDir, "dump.zip")

	// Build zip command arguments
	args := []string{"-r", zipPath, "."}
	for _, file := range sqlFiles {
		// Get relative path from tempDir
		relPath, err := filepath.Rel(tempDir, file)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}
		args = append(args, relPath)
	}

	// Add password if provided
	if zipPassword != "" {
		args = append([]string{"-P", zipPassword}, args...)
	}

	// Run zip command
	cmd := exec.Command("zip", args...)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating ZIP file: %s", output)
	}

	// Read the ZIP file and write to the writer
	zipFile, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("error opening ZIP file: %w", err)
	}
	defer zipFile.Close()

	if _, err := io.Copy(writer, zipFile); err != nil {
		return fmt.Errorf("error writing ZIP to output: %w", err)
	}

	return nil
}
