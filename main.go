package main

import (
	"fmt"
	"log"
	"google.golang.org/api/drive/v3"
	"github.com/bocon13/gdrivefix/app"
	"strings"
)

func traverse(srv *drive.Service, directoryId string, gen func(*drive.Service, int) (func(*drive.File), bool), depth int) {
	nextPageToken := ""
	fn, stop := gen(srv, depth)
	if stop {
		return
	}
	for {
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
			if i.MimeType == "application/vnd.google-apps.folder" {
				traverse(srv, i.Id, gen, depth+1)
			}
		}

		if r.NextPageToken == "" {
			break
		}
		nextPageToken = r.NextPageToken
	}
}

func print(depth int) (func(*drive.File), bool) {
	if depth > 2 {
		return nil, true
	}
	prefix := strings.Repeat("-", depth)
	return func(f *drive.File) {
		fmt.Println(prefix, f.Name, f.Id)
		//fmt.Println(i.Name, i.Id, i.MimeType)
		//fmt.Printf("%+v\n", i.Capabilities)
		//fmt.Printf("%+v\n", i.Permissions)
		//fmt.Printf("%+v\n", i.Parents)
		//fmt.Println("-----------------------")
	}, false
}

func setReadOnly(srv *drive.Service, depth int) (func(*drive.File), bool) {
	if depth > 1 {
		return nil, true
	}
	return func(f *drive.File) {
		fmt.Printf("%s (%s)\n", f.Name, f.Id)
		//FIXME set the owner to onlab admin account
		for _, p := range f.Permissions {
			if p.Type == "user" && strings.HasSuffix(p.EmailAddress, "onlab.us") {
				err := srv.Permissions.Delete(f.Id, p.Id).Do()
				if err != nil {
					fmt.Printf("Error removing permission for user %s (%s) on %s (%s): %s\n",
						p.DisplayName, p.EmailAddress, f.Name, f.Id, err)
				} else {
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

	rootDirId := "0B9D85zPc-_eQdzE3TTBxN3pBQTA"
	if rootDirId != "root" {
		fn, bool := setReadOnly(srv, 0)
		if !bool {
			f, err := srv.Files.Get(rootDirId).
				Fields("id, name, mimeType, capabilities, permissions, parents").
				Do()
			if err != nil {
				fmt.Println(err)
			} else {
				fn(f)
			}
		}
	}
	//traverse(srv, "root", print, 0)
	traverse(srv, rootDirId, setReadOnly, 0)
}