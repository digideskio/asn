// Copyright 2014-2015 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/apptimistco/asn/debug"
	"github.com/apptimistco/asn/debug/file"
)

const (
	ReposPS        = string(os.PathSeparator)
	ReposTopSz     = 2
	reposBridgeDN  = "bridge"
	reposBridgePat = AsnStr + ReposPS + reposBridgeDN + ReposPS
)

var (
	ErrAmbiguos = errors.New("Ambiguous USER or SHA")
	ErrNOENT    = errors.New("No such USER, SHA or FILE")
)

func ReadFromFile(rf ReadFromer, f *file.File) (err error) {
	dup, err := f.Dup()
	if err != nil {
		debug.Diag.Println(err)
		return err
	}
	defer dup.Close()
	var (
		v    Version
		id   Id
		blob Blob
	)
	if _, err = v.ReadFrom(dup); err != nil {
		debug.Diag.Println(err)
		return err
	}
	if _, err = id.ReadFrom(dup); err != nil {
		debug.Diag.Println(err)
		return err
	}
	if _, err = blob.ReadFrom(dup); err != nil {
		debug.Diag.Println(err)
		return err
	}
	_, err = rf.ReadFrom(dup)
	return err
}

type Repos struct {
	debug.Debug
	dn    string
	tmp   Tmp
	users Users
}

func (repos *Repos) Reset() {
	repos.tmp.Reset()
	repos.users.Reset()
	repos.dn = ""
}

func (repos *Repos) Set(v interface{}) error {
	dn, ok := v.(string)
	if !ok {
		return os.ErrInvalid
	}
	if err := repos.tmp.Set(dn); err != nil {
		return err
	}
	repos.dn = dn
	repos.Debug.Set(dn)
	repos.users.Set(dn)
	if err := repos.LoadUsers(); err != nil {
		repos.dn = ""
		repos.tmp.Reset()
		return err
	}
	return nil
}

// DePrefix strips leading repos directory from pathname
func (repos *Repos) DePrefix(pn string) string {
	return pn[len(repos.dn)+1:]
}

// Expand the stringified user key or blob sum to respective repos
// directory and file name.
func (repos *Repos) Expand(hex string, elements ...string) string {
	path := repos.Join(hex[:ReposTopSz], hex[ReposTopSz:])
	for _, x := range elements {
		path = filepath.Join(path, filepath.FromSlash(x))
	}
	return path
}

// Filter all REPOS/SHA files after epoch
func (repos *Repos) Filter(epoch time.Time,
	f func(fn string) error) (err error) {
	var (
		topdir, subdir *os.File
		topfis, subfis []os.FileInfo
	)
	topdir, err = os.Open(repos.dn)
	if err != nil {
		return
	}
	defer func() {
		if err == io.EOF {
			err = nil
		}
		topdir.Close()
		subdir.Close()
		topdir = nil
		subdir = nil
		topfis = nil
		subfis = nil
	}()
topdirloop:
	for {
		topfis, err = topdir.Readdir(16)
		if err != nil {
			break topdirloop
		}
	topfiloop:
		for _, topfi := range topfis {
			if !IsTopDir(topfi) {
				continue topfiloop
			}
			subfn := repos.Join(topfi.Name())
			if subdir, err = os.Open(subfn); err != nil {
				return
			}
		subdirloop:
			for {
				subfis, err = subdir.Readdir(16)
				if err == io.EOF {
					break subdirloop
				}
				if err != nil {
					return
				}
			subfiloop:
				for _, subfi := range subfis {
					if !IsBlob(subfi) {
						continue subfiloop
					}
					fn := repos.Join(topfi.Name(),
						subfi.Name())
					if epoch.IsZero() ||
						BlobTime(fn).After(epoch) {
						if err = f(fn); err != nil {
							return
						}
					}
				}
			}
		}
		topfis, err = topdir.Readdir(16)
	}
	return
}

func (repos Repos) FN2Ref(slogin, fn string) string {
	if strings.HasPrefix(fn, repos.dn) {
		fn = repos.DePrefix(fn)
	}
	if fn[ReposTopSz] != os.PathSeparator {
		return fn
	}
	topDN := fn[:ReposTopSz]
	fn = fn[ReposTopSz+1:]
	slash := strings.IndexByte(fn, os.PathSeparator)
	if slash < 0 {
		if IsHex(fn) && len(fn) > 14 {
			sumfn := topDN + fn
			return "$" + sumfn[:16]
		}
	} else if IsHex(fn[:slash]) && slash > 14 {
		suser := topDN + fn[:slash]
		if suser == slogin {
			return fn[slash+1:]
		} else {
			return "~" + suser[:16] + "/" + fn[slash+1:]
		}
	}
	return ""
}

func (repos *Repos) Glob(user, glob string) (m []string, err error) {
	fm, err := filepath.Glob(repos.Expand(user, glob))
	if err == nil {
		m = append(m, fm...)
		fm = nil
	}
	return
}

func (repos *Repos) Join(elements ...string) string {
	return repos.dn + ReposPS + filepath.Join(elements...)
}

func (repos *Repos) LoadUsers() error {
	var (
		topdir []os.FileInfo
		subdir []os.FileInfo
		err    error
	)
	defer func() {
		topdir = nil
		subdir = nil
	}()
	if topdir, err = ioutil.ReadDir(repos.dn); err != nil {
		return &Error{repos.dn, err.Error()}
	}
	for _, fi := range topdir {
		if fi.IsDir() && len(fi.Name()) == ReposTopSz {
			subdn := repos.Join(fi.Name())
			if subdir, err = ioutil.ReadDir(subdn); err != nil {
				return &Error{subdn, err.Error()}
			}
			for _, sub := range subdir {
				if sub.IsDir() && IsUser(sub.Name()) {
					suser := fi.Name() + sub.Name()
					user := repos.users.NewUserString(suser)
					err = user.cache.Load(repos.Join(user.DN()))
					if err != nil {
						return err
					}
				}
			}
			subdir = nil
		}
	}
	return nil
}

// lsm - Link and Send Message
func (repos *Repos) lsm(x Sender, sum *Sum, fn string, f *file.File,
	blob *Blob) {
	owner := repos.users.User(&blob.Owner)
	author := repos.users.User(&blob.Author)
	x.Send(&owner.key, f)
	LN(fn, repos.Join(owner.Join(AsnMessages, blob.FN(sum))))
	if author != owner {
		x.Send(&author.key, f)
		LN(fn, repos.Join(author.Join(AsnMessages, blob.FN(sum))))
	}
	if subscribers := owner.cache.Subscribers(); len(*subscribers) > 0 {
		for _, k := range *subscribers {
			sub := repos.users.User(&k)
			if sub != nil && sub != owner && sub != author {
				x.Send(&k, f)
				LN(fn, repos.Join(sub.Join(AsnMessages,
					blob.FN(sum))))
			}
		}
	}
}

// NewUser creates a cached user and repos directory
func (repos *Repos) NewUser(v interface{}) (user *User, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			repos.Diag(err, ":", v)
		}
	}()
	user = repos.UnsafeNewUser(v)
	return
}

func (repos *Repos) Open(fn string) (*file.File, error) {
	if !strings.HasPrefix(fn, repos.dn) {
		fn = repos.Join(fn)
	}
	return file.Open(fn)
}

// Process Approval and Removal sum lists.
func (repos *Repos) ProcSums(x Sender, f *file.File, blob *Blob) error {
	_, err := f.Seek(BlobNameOff+int64(len(blob.Name)), os.SEEK_SET)
	if err != nil {
		return err
	}
	author := repos.users.User(&blob.Author)
	for {
		var (
			sum    Sum
			target struct {
				fn    string
				f     *file.File
				fi    os.FileInfo
				v     Version
				id    Id
				blob  Blob
				owner *User
				stat  syscall.Stat_t
			}
		)
		if _, err = f.Read(sum[:]); err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		target.fn = repos.Join(sum.PN())
		if err = syscall.Stat(target.fn, &target.stat); err != nil {
			err = nil
			continue
		}
		if target.f, err = file.Open(target.fn); err != nil {
			return err
		}
		target.fi, _ = target.f.Stat()
		target.v.ReadFrom(target.f)
		target.id.ReadFrom(target.f)
		target.blob.ReadFrom(target.f)
		target.f.Close()
		target.f = nil
		target.owner = repos.users.User(&target.blob.Owner)
		if target.owner == nil {
			continue
		}
		if blob.Name == AsnApprovals {
			if target.stat.Nlink > 1 {
				repos.Diag(sum, "already linked")
				continue
			}
			if !author.MayApproveFor(target.owner) {
				repos.Diag(author.keystr[:8]+"...",
					"may not approve for",
					target.owner.keystr[:8]+"...")
				continue
			}
			if target.blob.Name != "" &&
				target.blob.Name != AsnMessages &&
				target.blob.Name != AsnMessages+"/" {
				repos.Diag("ignoring", target.blob.Name,
					": you may only approve", AsnMessages)
				continue
			}
			repos.lsm(x, &sum, target.fn, target.f, &target.blob)
		} else if blob.Name == AsnRemovals {
			if target.stat.Nlink == 1 {
				repos.Diag(sum, "has no links")
				continue
			}
			if !author.MayEdit(target.owner) {
				repos.Diag(author.keystr[:8]+"...",
					"may not remove from",
					target.owner.keystr[:8]+"...")
				continue
			}
			root := repos.Join(target.owner.Join())
			repos.Fixme("removal walk", root)
			filepath.Walk(root, func(path string, fi os.FileInfo,
				err error) error {
				if os.SameFile(target.fi, fi) {
					syscall.Unlink(path)
					repos.Diag("unlinked", path)
				}
				return err
			})
		}
	}
	return nil
}

func (repos *Repos) ParsePath(xn string) (user *User, fn string) {
	if strings.HasPrefix(xn, repos.dn) {
		xn = repos.DePrefix(xn)
	}
	if xn[ReposTopSz] != os.PathSeparator {
		fn = xn
		return
	}
	topDN := xn[:ReposTopSz]
	xn = xn[ReposTopSz+1:]
	slash := strings.IndexByte(xn, os.PathSeparator)
	var keystr string
	if IsHex(xn) {
		keystr = topDN + xn
	} else if slash > 0 && IsHex(xn[:slash]) {
		keystr = topDN + xn[:slash]
		fn = xn[slash+1:]
	} else {
		fn = topDN + xn
	}
	if keystr != "" {
		user = repos.users.UserString(keystr)
	}
	return
}

func (repos *Repos) Permission(owner, author *User, svc *ServiceKeys,
	name string) error {
	if bytes.Equal(author.key.Bytes(), svc.Admin.Pub.Encr.Bytes()) {
		return nil
	}
	if bytes.Equal(author.key.Bytes(), svc.Server.Pub.Encr.Bytes()) {
		return nil
	}
	if name == "" || name == AsnMessages || name == AsnMessages+"/" {
		return nil
	}
	if author.MayEdit(owner) {
		return nil
	}
	return os.ErrPermission
}

// Search the repos for the unique longest matching blob file.
func (repos *Repos) Search(x string) (match string, err error) {
	topdn := x[:ReposTopSz]
	subdfn := x[ReposTopSz:]
	lensubdfn := len(subdfn)
	topf, err := repos.Open(topdn)
	if err != nil {
		return
	}
	defer topf.Close()
	for {
		var names []string
		names, err = topf.Readdirnames(16)
		if len(names) > 0 {
			for _, name := range names {
				if len(name) >= lensubdfn &&
					name[:lensubdfn] == subdfn {
					if match != "" {
						match = ""
						err = ErrAmbiguos
						return
					}
					match = repos.Join(topdn, name)
				}
			}
		} else {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

// Store contents to file with name derrived from the calculated sum and
// forward through the given sender.  Returns the file sum on success;
// otherwise, error. If blob arg is nil, tee blob header from WriteToer.
func (repos *Repos) Store(x Sender, v Version, blob *Blob,
	wts ...WriteToer) (sum *Sum, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = r.(error)
			repos.Diag(debug.Depth(6), err)
		} else if err != nil {
			repos.Diag(debug.Depth(2), err)
		}
	}()
	var m io.Writer
	f := repos.tmp.New()
	defer repos.tmp.Free(f)
	h := sha512.New()
	defer h.Reset()
	if blob == nil {
		blob = NewBlob()
		defer blob.Free()
		m = io.MultiWriter(blob, f, h)
		if _, err = v.WriteTo(m); err != nil {
			return
		}
		if _, err = BlobId.Version(v).WriteTo(m); err != nil {
			return
		}
	} else {
		m = io.MultiWriter(f, h)
		if _, err = v.WriteTo(m); err != nil {
			return
		}
		if _, err = BlobId.Version(v).WriteTo(m); err != nil {
			return
		}
		if _, err = blob.WriteTo(m); err != nil {
			return
		}
	}
	for _, wt := range wts {
		if _, err = wt.WriteTo(m); err != nil {
			return
		}
	}
	sum = new(Sum)
	copy(sum[:], h.Sum([]byte{}))
	sumFN := repos.Join(sum.PN())
	if _, xerr := os.Stat(sumFN); os.IsNotExist(xerr) {
		LN(f.Name(), sumFN)
	} else {
		err = os.ErrExist
		return
	}
	owner := repos.User(&blob.Owner)
	author := repos.User(&blob.Author)
	switch {
	case blob.Name == AsnMark:
		err = ReadFromFile(owner.cache.Mark(), f)
		if err != nil {
			if err != io.EOF {
				return
			}
			err = nil
		}
		LN(sumFN, repos.Join(owner.Join(blob.Name)))
		repos.users.ForEachLoggedInUser(func(u *User) error {
			if u != owner {
				x.Send(&u.key, f)
			}
			return nil
		})
		// don't retain sum link as there is no need to recover a mark
		syscall.Unlink(sumFN)
	case blob.Name == AsnAuth:
		err = ReadFromFile(owner.cache.PubAuth(blob.Name), f)
		if err != nil {
			return
		}
		x.Send(Mirrors, f)
		LN(sumFN, repos.Join(owner.Join(blob.Name)))
	case blob.Name == AsnAuthor:
		err = ReadFromFile(owner.cache.PubEncr(blob.Name), f)
		if err != nil {
			return
		}
		x.Send(Mirrors, f)
		LN(sumFN, repos.Join(owner.Join(blob.Name)))
	case blob.Name == AsnEditors || blob.Name == AsnInvites ||
		blob.Name == AsnModerators || blob.Name == AsnSubscribers:
		err = ReadFromFile(owner.cache.PubEncrList(blob.Name), f)
		if err != nil {
			return
		}
		x.Send(Mirrors, f)
		LN(sumFN, repos.Join(owner.Join(blob.Name)))
	case blob.Name == AsnBridge, blob.Name == AsnBridge+"/":
		// don't link, just send to all invites
		for _, k := range *(owner.cache.Invites()) {
			x.Send(&k, f)
		}
		syscall.Unlink(sumFN)
	case blob.Name == "", blob.Name == AsnMessages,
		blob.Name == AsnMessages+"/":
		x.Send(Mirrors, f)
		moderators := owner.cache.Moderators()
		if len(*moderators) > 0 && !author.MayApproveFor(owner) {
			repos.Fixme("moderated")
			for _, k := range *moderators {
				x.Send(&k, f)
			}
			return
		}
		repos.Fixme("unmoderated")
		repos.lsm(x, sum, sumFN, f, blob)
	case strings.HasSuffix(blob.Name, "/"):
		x.Send(Mirrors, f)
		LN(sumFN, repos.Join(owner.Join(blob.Name, blob.FN(sum))))
	case blob.Name == AsnApprovals || blob.Name == AsnRemovals:
		if err = repos.ProcSums(x, f, blob); err != nil {
			return
		}
		x.Send(Mirrors, f)
		LN(sumFN, repos.Join(owner.Join(blob.Name, blob.FN(sum))))
	default:
		fn := repos.Join(owner.Join(blob.Name))
		if _, xerr := os.Stat(fn); xerr == nil {
			if BlobTime(fn).After(blob.Time) {
				return // don't link or mirror older blob
			}
			syscall.Unlink(fn)
		}
		x.Send(Mirrors, f)
		LN(sumFN, fn)
	}
	return
}

// UnsafeNewUser will panic on error so the calling function must recover.
func (repos *Repos) UnsafeNewUser(v interface{}) (user *User) {
	var i int
	repos.users.Lock()
	defer repos.users.Unlock()
	n := len(repos.users.l)
	switch t := v.(type) {
	case string:
		i = sort.Search(n, func(i int) bool {
			return repos.users.l[i].String() >= t
		})
		user = NewUserString(t)
	case *PubEncr:
		i = sort.Search(n, func(i int) bool {
			return bytes.Compare(repos.users.l[i].key.Bytes(),
				t.Bytes()) >= 0
		})
		user = NewUserKey(t)
	default:
		panic(os.ErrInvalid)
	}
	if i == n {
		repos.users.l = append(repos.users.l, user)
	} else {
		repos.users.l = append(repos.users.l[:i],
			append([]*User{user},
				(repos.users.l[i:])...)...)
	}
	dn := repos.Join(user.dn)
	if _, err := os.Stat(dn); err != nil {
		if err = MkdirAll(dn); err != nil {
			user = nil
			panic(err)
		}
	}
	return user
}

// User search and create if not found
func (repos *Repos) User(k *PubEncr) *User {
	{
		var emptyPubEncr PubEncr
		if bytes.Equal(k.Bytes(), emptyPubEncr.Bytes()) {
			repos.Diag(debug.Depth(2), "User with null key")
			repos.Diag(debug.Depth(3), "User with null key")
			repos.Diag(debug.Depth(4), "User with null key")
		}
	}
	user := repos.users.User(k)
	if user == nil {
		user = repos.UnsafeNewUser(k)
	}
	return user
}
