package ssh

import (
	"strings"
	"fmt"
	"os"
	"io"
	"path"
	"errors"
)
// assumes remote dst is directory
func scp_local_remote(auth *Auth, src, user, ip, dst string) error {
	client , err := NewNativeClient(user, ip, "", 22, auth)
	if err != nil {	return err }

	reader, err := os.Open(src)
	if err != nil {	return err }

	err = client.CopyFile(reader, path.Join(dst, path.Base(src)), 0644)
	if err != nil {	return err }

	fmt.Printf("Copied %s to %s@%s:%s\n", src, user, ip, dst)
	return nil
}
func scp_remote_local(auth *Auth, user, ip, src, dst string) error {
	var client Client
	var err error
	client, err = NewNativeClient(user, ip, "", 22, auth)
	if err != nil { return err }
	stdout, _, err := client.Start("cat " + src)
	if err != nil { return err }
	finfo, err := os.Stat(dst)
	if err != nil { return err }
	if finfo.IsDir() {
		dst = path.Join(dst, path.Base(src))
	}
	file, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil { return err }
	defer file.Close()
	client.Wait()
	io.Copy(file, stdout)
	fmt.Printf("Copied %s@%s:%s to %s\n", user, ip, src, dst)
	return nil
}
func scp_remote_remote(auth *Auth, user1, ip1, src, user2, ip2, dst string) error {
	// At the moment this library is not able to have two different sessions at the same time (hangs)
	//   once that's fixed then it makes sense to implement this.
	return errors.New("Copying from remote to remote is not implemented yet.")
}
func scp_local_local(src, dst string) error {
	finfo, err := os.Stat(dst)
	if err != nil {	return err }
	if finfo.IsDir() {
		dst = path.Join(dst, path.Base(src))
	}
	reader, err := os.Open(src)
	if err != nil {	return err }
	destfile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {	return err }
	defer destfile.Close()
	io.Copy(destfile, reader)
	fmt.Printf("Copied %s to %s\n", src, dst)
	return nil
}

type remotePath struct { user, host, path string }
// copy files with scp format, eg: user@host:/path/to/remote/file
func Scp(auth *Auth, recursive bool, files ...string) error {
	if recursive {
		fmt.Println(files)
		return errors.New("Recursive copy is not implemented yet")
	}
	dst := files[len(files) - 1]
	dstIsRemote := strings.Index(dst, ":") >= 0
	rDst := remotePath{}
	if dstIsRemote {
		splitAt := strings.Split(dst, "@")
		rDst.user = splitAt[0]
		splitColon := strings.Split(splitAt[1], ":")
		rDst.host = splitColon[0]
		rDst.path = splitColon[1]
	}
	var err error
	for i, f := range files {
		if i == len(files) - 1 {break}
		remote := strings.Split(f, ":")
		switch len(remote) {
		case 1:
			if dstIsRemote {
				err = scp_local_remote(auth, f, rDst.user, rDst.host, rDst.path)
			} else {
				err = scp_local_local(f, dst)
			}
		case 2:
			path := remote[1]
			userHost := strings.Split(remote[0], "@")
			user := userHost[0]
			host := userHost[1]
			if dstIsRemote{
				err = scp_remote_remote(auth, user, host, path, rDst.user, rDst.host, rDst.path)
			} else {
				err = scp_remote_local(auth, user, host, path, dst)
			}
		default:
			return errors.New("Invalid path: " + f)
		}
		if err != nil { return err }
	}
	return nil
}