package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bocon13/gdrivefix/app"
	"google.golang.org/api/drive/v3"
)

// Traverse the specified file/directory, starting by generating a function at depth 0 and applying it to the file
func traverse(srv *drive.Service, directoryId string, gen func(*drive.Service, int) (func(*drive.File), bool)) {
	fn, bool := gen(srv, 0)
	if bool {
		return
	}
	f, err := srv.Files.Get(directoryId).
		Fields("id, name, mimeType, capabilities, permissions, parents").
		Do()
	if err != nil {
		fmt.Println(err)
		return
	}
	fn(f)
	_traverse(srv, directoryId, gen, 1)
}

func _traverse(srv *drive.Service, directoryId string, gen func(*drive.Service, int) (func(*drive.File), bool), depth int) {
	// Generate the function for the appropriate depth to run on each node
	fn, stop := gen(srv, depth)
	if stop {
		return
	}
	nextPageToken := ""
	for {
		// Find all children of a parent directory, results will be paged (hence the loop)
		c := srv.Files.List().
			Q(fmt.Sprintf("'%s' in parents", directoryId)).
			Fields("nextPageToken", "files(id, name, mimeType, capabilities, permissions, parents)")
		if nextPageToken != "" {
			c = c.PageToken(nextPageToken)
		}
		r, err := c.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		for _, i := range r.Files {
			fn(i)
			// Type == Google Drive Folder
			if i.MimeType == "application/vnd.google-apps.folder" {
				_traverse(srv, i.Id, gen, depth+1)
			}
		}

		if r.NextPageToken == "" {
			break
		}
		nextPageToken = r.NextPageToken
	}
}

func setReadOnly(srv *drive.Service, depth int) (func(*drive.File), bool) {
	return func(f *drive.File) {
		fmt.Printf("%s (%s)\n", f.Name, f.Id)
		srv.Permissions.Create(f.Id, &drive.Permission{
			EmailAddress: "admin@onlab.us",
			Role:         "owner",
			Type:         "user",
		}).TransferOwnership(true).Do()

		fmt.Println("------------------------")
	}, false
}

func main() {
	client := app.Client()

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	if len(os.Args) < 2 {
		fmt.Println("TODO: USAGE")
		return
	}

	if os.Args[1] == "-l" {
		// Recursively list the directories starting at "My Drive" to the specified depth
		maxRecursiveDepth, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		traverse(srv, "root", func(srv *drive.Service, depth int) (func(*drive.File), bool) {
			if depth > maxRecursiveDepth {
				return nil, true
			}
			prefix := strings.Repeat("-", depth)
			return func(f *drive.File) {
				fmt.Printf("%s %s (%s)\n", prefix, f.Name, f.Id)
			}, false
		})
	} else if os.Args[1] == "-u" {
		// Recursively fix permissions starting at the specified file/directory
		rootDirId := os.Args[2]
		traverse(srv, rootDirId, setReadOnly)
	}
}
