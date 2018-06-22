package main

import (
	"errors"
	"fmt"
	"os"
	pathutil "path"
	"strconv"

	"github.com/drone-plugins/drone-gcs-cache/storage/gcs"
	"github.com/drone/drone-cache-lib/storage"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	version = "0.0.0"
	build   = "0"
)

func main() {
	app := cli.NewApp()
	app.Name = "cache plugin"
	app.Usage = "cache plugin"
	app.Version = fmt.Sprintf("%s+%s", version, build)
	app.Action = run
	app.Flags = []cli.Flag{
		// Cache information

		cli.StringFlag{
			Name:   "filename",
			Usage:  "Filename for the cache",
			EnvVar: "PLUGIN_FILENAME",
		},
		cli.StringFlag{
			Name:   "bucket",
			Usage:  "bucket",
			EnvVar: "PLUGIN_BUCKET",
		},
		cli.StringFlag{
			Name:   "path",
			Usage:  "path",
			EnvVar: "PLUGIN_PATH",
		},
		cli.StringFlag{
			Name:   "fallback_path",
			Usage:  "fallback_path",
			EnvVar: "PLUGIN_FALLBACK_PATH",
		},
		cli.StringSliceFlag{
			Name:   "mount",
			Usage:  "cache directories",
			EnvVar: "PLUGIN_MOUNT",
		},
		cli.BoolFlag{
			Name:   "rebuild",
			Usage:  "rebuild the cache directories",
			EnvVar: "PLUGIN_REBUILD",
		},
		cli.BoolFlag{
			Name:   "restore",
			Usage:  "restore the cache directories",
			EnvVar: "PLUGIN_RESTORE",
		},
		cli.BoolFlag{
			Name:   "flush",
			Usage:  "flush the cache",
			EnvVar: "PLUGIN_FLUSH",
		},
		cli.StringFlag{
			Name:   "flush_age",
			Usage:  "flush cache files older than # days",
			EnvVar: "PLUGIN_FLUSH_AGE",
			Value:  "30",
		},
		cli.StringFlag{
			Name:   "flush_path",
			Usage:  "path to search for flushable cache files",
			EnvVar: "PLUGIN_FLUSH_PATH",
		},
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug plugin output",
			EnvVar: "PLUGIN_DEBUG",
		},

		// Build information (for setting defaults)

		cli.StringFlag{
			Name:   "repo.owner",
			Usage:  "repository owner",
			EnvVar: "DRONE_REPO_OWNER",
		},
		cli.StringFlag{
			Name:   "repo.name",
			Usage:  "repository name",
			EnvVar: "DRONE_REPO_NAME",
		},
		cli.StringFlag{
			Name:   "repo.branch",
			Value:  "master",
			Usage:  "repository default branch",
			EnvVar: "DRONE_REPO_BRANCH",
		},
		cli.StringFlag{
			Name:   "commit.branch",
			Value:  "master",
			Usage:  "git commit branch",
			EnvVar: "DRONE_COMMIT_BRANCH",
		},

		// GCS information

		cli.StringFlag{
			Name:   "json_key",
			Usage:  "google service account json key",
			EnvVar: "PLUGIN_JSON_KEY,GCS_CACHE_JSON_KEY",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// Set the logging level
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	// Determine the mode for the plugin
	rebuild := c.Bool("rebuild")
	restore := c.Bool("restore")
	flush := c.Bool("flush")

	if isMultipleModes(rebuild, restore, flush) {
		return errors.New("Must use a single mode: rebuild, restore or flush")
	} else if !rebuild && !restore && !flush {
		return errors.New("No action specified")
	}

	var mode string
	var mount []string

	if rebuild {
		// Look for the mount points to rebuild
		mount = c.StringSlice("mount")

		if len(mount) == 0 {
			return errors.New("No mounts specified")
		}

		mode = RebuildMode
	} else if flush {
		mode = FlushMode
	} else {
		mode = RestoreMode
	}

	// Get the root path prefix to place the cache files
	root := c.GlobalString("bucket")

	// Get the path to place the cache files
	path := c.GlobalString("path")

	// Defaults to <owner>/<repo>/<branch>/
	if len(path) == 0 {
		log.Info("No path specified. Creating default")

		path = fmt.Sprintf(
			"%s/%s/%s",
			c.String("repo.owner"),
			c.String("repo.name"),
			c.String("commit.branch"),
		)

		path = prefixRoot(root, path)
	}

	// Get the fallback path to retrieve the cache files
	fallbackPath := c.GlobalString("fallback_path")

	// Defaults to <owner>/<repo>/master/
	if len(fallbackPath) == 0 {
		log.Info("No fallback_path specified. Creating default")

		fallbackPath = fmt.Sprintf(
			"%s/%s/%s",
			c.String("repo.owner"),
			c.String("repo.name"),
			c.String("repo.branch"),
		)

		fallbackPath = prefixRoot(root, fallbackPath)
	}

	// Get the flush path to flush the cache files from
	flushPath := c.GlobalString("flush_path")

	// Defaults to <owner>/<repo>/
	if len(flushPath) == 0 {
		log.Info("No flush_path specified. Creating default")

		flushPath = fmt.Sprintf(
			"%s/%s",
			c.String("repo.owner"),
			c.String("repo.name"),
		)

		flushPath = prefixRoot(root, flushPath)
	}

	// Get the filename
	filename := c.GlobalString("filename")

	if len(filename) == 0 {
		log.Info("No filename specified. Creating default")

		filename = "archive.tar"
	}

	s, err := gcStorage(c)

	if err != nil {
		return err
	}

	flushAge, err := strconv.Atoi(c.String("flush_age"))

	if err != nil {
		return err
	}

	p := &Plugin{
		Filename:     filename,
		Path:         path,
		FallbackPath: fallbackPath,
		FlushPath:    flushPath,
		Mode:         mode,
		FlushAge:     flushAge,
		Mount:        mount,
		Storage:      s,
	}

	return p.Exec()
}

func gcStorage(c *cli.Context) (storage.Storage, error) {

	// Get the access credentials
	jsonkey := c.String("json_key")

	return gcs.New(&gcs.Options{
		JSONKey: jsonkey,
	})
}

func isMultipleModes(bools ...bool) bool {
	var b bool
	for _, v := range bools {
		if b && b == v {
			return true
		}

		if v {
			b = true
		}
	}

	return false
}

func prefixRoot(root, path string) string {
	return pathutil.Clean(fmt.Sprintf("/%s/%s", root, path))
}
