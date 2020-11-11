package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/yaml.v2"
)

func main() {
	var (
		repoURL       string
		trackedFolder string
		branch        string
	)
	flag.StringVar(&repoURL, "repo", "dvc-uploader-test", "url or path of the git repository")
	flag.StringVar(&trackedFolder, "trackedFolder", "dataset.dvc", "Path to the file .dvc file that tracks the folder content")
	flag.StringVar(&branch, "branch", "master", "git branch to add data")
	flag.Parse()

	files := flag.Args()
	log.Printf("Files to upload %s", files)

	// if repoURL == "" {
	// 	panic("you must provide a repository URL with -repo")
	// }
	// if fileToUpload == "" {
	// 	panic("you must provide a file with -file flag")
	// }
	// if trackedFolder == "" {
	// 	panic("you must provide a tracked folder with -trackedFolder")
	// }

	fileName := flag.Arg(0)

	uploadFile, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	hash := md5.New()
	_, err = io.Copy(hash, uploadFile)
	if err != nil {
		panic(err)
	}

	fileMD5 := hex.EncodeToString(hash.Sum(nil))
	log.Printf("File hash %s", fileMD5)

	storage := memory.NewStorage()
	fs := memfs.New()
	repo, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		panic(err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		panic(err)
	}

	// err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch)})
	// if err != nil {
	// 	panic(err)
	// }

	// repo, err := git.PlainOpen("./example-get-started")
	// repo, err := git.PlainOpen("./dvc-uploader-test")
	// if err != nil {
	// 	panic(err)
	// }
	// rev, err := repo.ResolveRevision("HEAD")
	// if err != nil {
	// 	panic(err)
	// }

	// commit, err := repo.CommitObject(*rev)
	// if err != nil {
	// 	panic(err)
	// }

	config, err := wt.Filesystem.Open(".dvc/config")
	// config, err := commit.File(".dvc/config")
	if err != nil {
		panic(err)
	}
	defer config.Close()

	// creader, err := config.Reader()
	// if err != nil {
	// 	panic(err)
	// }

	gconf, err := NewGlobalConfig(config)
	if err != nil {
		panic(err)
	}

	defaultRemote, err := gconf.DefaultRemote()
	if err != nil {
		panic(fmt.Errorf("Cannot read the default remote, %w", err))
	}

	log.Printf("Using remote %s", defaultRemote)

	_, err = uploadFile.Seek(0, 0)
	if err != nil {
		panic(err)
	}

	err = defaultRemote.Upload(context.Background(), fileMD5, uploadFile)
	if err != nil {
		panic(err)
	}

	log.Print("File uploaded to remote")

	trackedFolderFile, err := wt.Filesystem.Open(trackedFolder)
	// trackedFolderFile, err := commit.File(trackedFolder)
	if err != nil {
		panic(fmt.Errorf("Cannot find tracked folder file %s, %w", trackedFolder, err))
	}

	// trackedFolderReader, err := trackedFolderFile.Reader()
	// if err != nil {
	// 	panic(err)
	// }

	decoder := yaml.NewDecoder(trackedFolderFile)

	dvcFile := &DVCFile{}
	err = decoder.Decode(dvcFile)
	if err != nil {
		panic(err)
	}

	log.Printf("Uploading to folder %+v", dvcFile)

	content, err := defaultRemote.Download(context.Background(), dvcFile.Outs[0].MD5)
	if err != nil {
		panic(err)
	}

	jDecoder := json.NewDecoder(content)
	dirContent := []DVCDirListItem{}
	err = jDecoder.Decode(&dirContent)
	if err != nil {
		panic(err)
	}

	dirContent = append(dirContent, DVCDirListItem{
		MD5:     fileMD5,
		RelPath: fileName,
	})

	newTrackedFolder, err := wt.Filesystem.Create(trackedFolder)
	if err != nil {
		panic(err)
	}

	sum := md5.New()
	jEncoder := json.NewEncoder(io.MultiWriter(newTrackedFolder, sum))
	jEncoder.Encode(dirContent)
	newTrackedFolder.Close()

	dirFile, err := wt.Filesystem.Open(trackedFolder)
	dirMD5 := hex.EncodeToString(sum.Sum(nil))
	err = defaultRemote.Upload(context.Background(), dirMD5, dirFile)
	if err != nil {
		panic(err)
	}
	dirFile.Close()

	log.Printf("Uploaded new dir %s", dirMD5)

	_, err = wt.Add(trackedFolder)
	if err != nil {
		panic(err)
	}

	finalCommit, err := wt.Commit("Update file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "dvc uploader",
			Email: "invalid@nope.com",
		},
	})
	if err != nil {
		panic(err)
	}
	log.Print("Committed changes ", finalCommit.String())

	err = repo.Push(&git.PushOptions{})
	if err != nil {
		panic(err)
	}

	log.Print("pushed to git")
}

type GlobalConfig struct {
	file *ini.File
}

func NewGlobalConfig(reader io.Reader) (*GlobalConfig, error) {
	cini, err := ini.InsensitiveLoad(reader)
	if err != nil {
		return nil, err
	}
	return &GlobalConfig{
		file: cini,
	}, nil
}

type Core struct {
	Remote string
}

type Remote interface {
	GetURL() string
	Upload(ctx context.Context, md5 string, data io.Reader) error
	Download(ctx context.Context, md5 string) (io.ReadCloser, error)
}

type HttpRemote struct {
	Remote
	URL string
}

type LocalRemote struct {
	URL string
}

func (remote *LocalRemote) Upload(ctx context.Context, path string, data io.Reader) error {
	remotePath := filepath.Join(remote.URL, path[:2], path[2:])
	err := os.MkdirAll(filepath.Dir(remotePath), 0700)
	if err != nil {
		return fmt.Errorf("Cannot create directory, %w", err)
	}

	file, err := os.Create(remotePath)
	if err != nil {
		return fmt.Errorf("Cannot create file, %w", err)
	}

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("Could not write file, %w", err)
	}
	return nil
}

func (remote *LocalRemote) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.Join(remote.URL, path[:2], path[2:]))
	if err != nil {
		return nil, fmt.Errorf("Cannot open file, %w", err)
	}
	return file, nil
}

func (remote *LocalRemote) GetURL() string {
	return remote.URL
}

func (config *GlobalConfig) Core() (*Core, error) {
	s, err := config.file.GetSection("core")
	if err != nil {
		return nil, err
	}

	core := &Core{}

	err = s.MapTo(core)
	if err != nil {
		return nil, err
	}
	return core, nil
}

func (config *GlobalConfig) Remotes() (map[string]Remote, error) {
	remotes := make(map[string]Remote)
	for _, sectionName := range config.file.SectionStrings() {
		matches := regexp.MustCompile("remote \"(.*)\"").FindStringSubmatch(sectionName)
		if len(matches) != 2 {
			continue
		}
		remoteName := matches[1]

		section, err := config.file.GetSection(sectionName)
		if err != nil {
			return nil, err
		}

		urlKey, err := section.GetKey("url")
		if err != nil {
			return nil, err
		}
		url := urlKey.String()

		switch {
		case strings.HasPrefix(url, "https://"):
			remote := &HttpRemote{}
			err = section.MapTo(remote)
			if err != nil {
				return nil, err
			}
			remotes[remoteName] = remote
		default:
			if !filepath.IsAbs(url) {
				panic(fmt.Errorf("Path must be absolute, %s", url))
			}

			remote := &LocalRemote{}
			err = section.MapTo(remote)
			if err != nil {
				return nil, err
			}
			remotes[remoteName] = remote
		}

	}
	return remotes, nil
}

func (config *GlobalConfig) Remote(name string) (Remote, error) {
	remotes, err := config.Remotes()
	if err != nil {
		return nil, err
	}

	remote, ok := remotes[name]
	if !ok {
		return nil, fmt.Errorf("Not found")
	}

	return remote, nil
}

func (config *GlobalConfig) DefaultRemote() (Remote, error) {
	core, err := config.Core()
	if err != nil {
		return nil, err
	}
	return config.Remote(core.Remote)
}

type DVCFile struct {
	Outs []DVCOut
}

type DVCOut struct {
	MD5  string
	Path string
}

type DVCDirListItem struct {
	MD5     string
	RelPath string
}
