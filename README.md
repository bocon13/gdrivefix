# gdrivefix
This program does two things:
1. Looks for documents that you have shared with @onlab.us address and updates the email to @opennetworking.org
2. Sets the role to "reader" for everyone that is not the owner.

A minor enhancement that I might add soon would be to transfer the ownership to an admin account that will remain in the ON.Lab domain.

## Setup

Install Go

Run `go get -u github.com/bocon13/gdrivefix`

Then `cd $GOPATH/src/github.com/bocon13/gdrivefix/`
`

Follow steps #1 and #2 here:
https://developers.google.com/drive/v3/web/quickstart/go

Make sure to rename the file you download to: client_secret.json and put it
in the same directory as main.go

## Usage

First, find the target directory ID by running:

```go run main.go -l 1```

The `-l` means list directories, and the `1` is the
recursive depth.

This command will print out the directories and files in
your "My Drive".

Next, take the ID (which is in the parenthesis) and run

```go run main.go -u [ID]```

It will print out the permission actions taken.

Note: The first time that you run one of these commands
it will prompt you to authenticate with Google: go to the URL,
log in with your ON.Lab email, and copy the code.
Then, paste the code into your terminal and hit return.