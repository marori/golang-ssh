package ssh

import (
	"strings"
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"path"
	"errors"
)
// assumes remote dst is directory
func scp_local_remote(auth *Auth, recursive bool, src, user, ip, dst string) error {
	client , err := NewNativeClient(user, ip, "", 22, auth)
	if err != nil {	return err }

	baselocal /Users/jdias/workspaces/github.concur.com/jdias/puppetinit-docker
	   /hieradata/.DS_Store
	directory -> /etc/puppetlabs/code/hieradata/aced
	D0755 0 /etc/puppetlabs/code/hieradata/aced/Users/jdias/workspaces/github.concur.com/jdias/puppetinit-docker/hieradata

	fmt.Println()
	fmt.Println(src)
	fmt.Println(dst)
	reader, err := os.Open(src)
	if err != nil {	return err }
	dst = path.Join(dst, src)
	err = client.CopyFile(reader, recursive, dst, 0644)
	if err != nil {	return err }

//	fmt.Printf("Copied %s to %s@%s:%s\n", src, user, ip, dst)
	return nil
}
func scp_remote_local(auth *Auth, recursive bool, user, ip, src, dst string) error {
	if recursive { return errors.New("recursive remote to local not implemented yet") }
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
func scp_remote_remote(auth *Auth, recursive bool, user1, ip1, src, user2, ip2, dst string) error {
	// At the moment this library is not able to have two different sessions at the same time (hangs)
	//   once that's fixed then it makes sense to implement this.
	return errors.New("Copying from remote to remote is not implemented yet.")
}
func scp_local_local(recursive bool, src, dst string) error {
	if recursive {
		err := os.MkdirAll(path.Join(dst, path.Dir(src)), 0755)
		if err != nil { return err }
	}
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
func isDir(f string) bool {
	i, e := os.Stat(f)
	if e != nil { return false }
	return i.IsDir()
}

func getSubDirs(dirname string) ([]string, error) {
	fis, err := ioutil.ReadDir(dirname)
	filelist := make([]string, 0)
	if err != nil {
		return nil, errors.New(err.Error()) }
	for _, fi := range fis {
		if fi.IsDir() {
			fs, e := getSubDirs(path.Join(dirname, fi.Name()))
			if e != nil { return nil, e }
			filelist = append(filelist, fs...)
		} else {
			filelist = append(filelist, path.Join(dirname, fi.Name()))
		}
	}
	return filelist, nil
}

// finds all files under directories for recursive copy
func flattenFiles(files []string) ([]string, error){
	res := make([]string, 0)
	for _, f := range files {
		if isDir(f) {
			fs, e := getSubDirs(f)
			if e != nil { return nil, e }
			res = append(res, fs...)
		} else {
			res = append(res, f)
		}
	}
	return res, nil
}

type remotePath struct { user, host, path string }
// copy files with scp format, eg: user@host:/path/to/remote/file
func Scp(auth *Auth, recursive bool, files ...string) error {
	lf := len(files)
	srcFiles := files[:lf - 1]
	dst := files[len(files) - 1]
	var err error
	if recursive {
		srcFiles, err = flattenFiles(srcFiles)
		if err != nil { return err }
	}
	dstIsRemote := strings.Index(dst, ":") >= 0
	rDst := remotePath{}
	if dstIsRemote {
		splitAt := strings.Split(dst, "@")
		rDst.user = splitAt[0]
		splitColon := strings.Split(splitAt[1], ":")
		rDst.host = splitColon[0]
		rDst.path = splitColon[1]
	}
	for _, f := range srcFiles {
		remote := strings.Split(f, ":")
		switch len(remote) {
		case 1:
			if dstIsRemote {
				err = scp_local_remote(auth, recursive, f, rDst.user, rDst.host, rDst.path)
			} else {
				err = scp_local_local(recursive, f, dst)
			}
		case 2:
			path := remote[1]
			userHost := strings.Split(remote[0], "@")
			user := userHost[0]
			host := userHost[1]
			if dstIsRemote{
				err = scp_remote_remote(auth, recursive, user, host, path, rDst.user, rDst.host, rDst.path)
			} else {
				err = scp_remote_local(auth, recursive, user, host, path, dst)
			}
		default:
			return errors.New("Invalid path: " + f)
		}
		if err != nil { return err }
	}
	return nil
}