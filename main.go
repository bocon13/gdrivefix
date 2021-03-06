package main

import (
	"fmt"
	"log"
	"google.golang.org/api/drive/v3"
	"github.com/bocon13/gdrivefix/app"
	"strings"
	"os"
	"strconv"
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
		//FIXME set the owner to onlab admin account
		for _, p := range f.Permissions {
			if p.Type == "user" && strings.HasSuffix(p.EmailAddress, "onlab.us") {
				// Remove @onlab.us account permission
				// Note: This will currently fail for the file owner
				err := srv.Permissions.Delete(f.Id, p.Id).Do()
				if err != nil {
					fmt.Printf("Error removing permission for user %s (%s) on %s (%s): %s\n",
						p.DisplayName, p.EmailAddress, f.Name, f.Id, err)
				} else {
					// Create read-only onf.org permission for the same user
					newEmail := strings.Replace(p.EmailAddress, "onlab.us", "opennetworking.org", 1)
					_, err = srv.Permissions.Create(f.Id, &drive.Permission{
						EmailAddress: newEmail,
						Role:         "reader",
						Type: "user",
					}).Do()
					if err != nil {
						fmt.Printf("Error creating permission for user %s (%s) on %s (%s): %s\n",
							p.DisplayName, newEmail, f.Name, f.Id, err)
					} else {
						fmt.Printf("Added %s (%s) as reader on %s", p.DisplayName, newEmail, f.Name)
					}
				}
			} else if p.Role != "reader" {
				// Update non-read-only permission to read-only
				// Note: This will currently fail for the file owner
				_, err := srv.Permissions.Update(f.Id, p.Id, &drive.Permission{
					Role: "reader",
				}).Do()
				if err != nil {
					fmt.Printf("Error updating permission for user %s (%s) on %s (%s): %s\n",
						p.DisplayName, p.EmailAddress, f.Name, f.Id, err)
				} else {
					fmt.Printf("Set %s to reader on %s", p.DisplayName, f.Name)
				}
			} else {
				fmt.Printf("Ignoring %s (%s) on %s\n", p.DisplayName, p.EmailAddress, f.Name)
			}
		}
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