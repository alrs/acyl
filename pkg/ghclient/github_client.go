package ghclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	ghTimeout = 2 * time.Minute
)

// BranchInfo includes the information about a specific branch of a git repo
type BranchInfo struct {
	Name string
	SHA  string
}

// CommitStatus describes a status associated with a git commit
type CommitStatus struct {
	Context     string
	Status      string
	Description string
	TargetURL   string
}

// RepoClient describes an object capable of operating on git repositories
type RepoClient interface {
	GetBranch(context.Context, string, string) (BranchInfo, error)
	GetBranches(context.Context, string) ([]BranchInfo, error)
	GetTags(context.Context, string) ([]BranchInfo, error)
	SetStatus(context.Context, string, string, *CommitStatus) error
	GetPRStatus(context.Context, string, uint) (string, error)
	GetCommitMessage(context.Context, string, string) (string, error)
	GetFileContents(ctx context.Context, repo string, path string, ref string) ([]byte, error)
	GetDirectoryContents(ctx context.Context, repo, path, ref string) (map[string]FileContents, error)
}

// GitHubClient is an object that interacts with the GitHub API
type GitHubClient struct {
	c *github.Client
}

// NewGitHubClient returns a GitHubClient instance that authenticates using
// the personal access token provided
func NewGitHubClient(token string) *GitHubClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GitHubClient{
		c: github.NewClient(tc),
	}
}

var _ RepoClient = &GitHubClient{}

// GetBranch gets the information for a specific branch
func (ghc *GitHubClient) GetBranch(ctx context.Context, repo, branch string) (BranchInfo, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return BranchInfo{}, fmt.Errorf("malformed repo: %v", repo)
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	b, _, err := ghc.c.Repositories.GetBranch(ctx, rs[0], rs[1], branch)
	if err != nil {
		return BranchInfo{}, fmt.Errorf("error getting branch: %v", err)
	}
	return BranchInfo{
		Name: branch,
		SHA:  *b.Commit.SHA,
	}, nil
}

// GetBranches returns the extant branches for repo (assumed to be "owner/repo-name")
func (ghc *GitHubClient) GetBranches(ctx context.Context, repo string) ([]BranchInfo, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return nil, fmt.Errorf("malformed repo: %v", repo)
	}
	output := []BranchInfo{}
	lopt := github.ListOptions{PerPage: 100}
	for {
		ctx, cf := context.WithTimeout(ctx, ghTimeout)
		defer cf()
		r, resp, err := ghc.c.Repositories.ListBranches(ctx, rs[0], rs[1], &lopt)
		if err != nil {
			return nil, fmt.Errorf("error listing branches: %v", err)
		}
		for _, b := range r {
			output = append(output, BranchInfo{
				Name: *b.Name,
				SHA:  *b.Commit.SHA,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}
	return output, nil
}

// GetTags returns the extant tags for repo (assumed to be "owner/repo-name")
func (ghc *GitHubClient) GetTags(ctx context.Context, repo string) ([]BranchInfo, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return nil, fmt.Errorf("malformed repo: %v", repo)
	}
	output := []BranchInfo{}
	lopt := github.ListOptions{PerPage: 100}
	for {
		ctx, cf := context.WithTimeout(ctx, ghTimeout)
		defer cf()
		r, resp, err := ghc.c.Repositories.ListTags(ctx, rs[0], rs[1], &lopt)
		if err != nil {
			return nil, fmt.Errorf("error listing branches: %v", err)
		}
		for _, b := range r {
			output = append(output, BranchInfo{
				Name: *b.Name,
				SHA:  *b.Commit.SHA,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}
	return output, nil
}

// SetStatus creates or updates a status on repo at commit sha
// Note that the github client CreateStatus function will bail if the context was canceled. It is often recommended to pass a fresh context to this function.
func (ghc *GitHubClient) SetStatus(ctx context.Context, repo string, sha string, status *CommitStatus) error {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return fmt.Errorf("malformed repo: %v", repo)
	}
	gs := &github.RepoStatus{
		State:       &status.Status,
		TargetURL:   &status.TargetURL,
		Description: &status.Description,
		Context:     &status.Context,
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	_, _, err := ghc.c.Repositories.CreateStatus(ctx, rs[0], rs[1], sha, gs)
	return err
}

// GetPRStatus returns the status of a PR on a repo
func (ghc *GitHubClient) GetPRStatus(ctx context.Context, repo string, pr uint) (string, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return "", fmt.Errorf("malformed repo: %v", repo)
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	pullreq, _, err := ghc.c.PullRequests.Get(ctx, rs[0], rs[1], int(pr))
	if err != nil {
		return "", err
	}
	if pullreq == nil || pullreq.State == nil {
		return "", fmt.Errorf("pull request is nil")
	}
	return *pullreq.State, nil
}

// GetCommitMessage returns the git message associated with a particular commit
func (ghc *GitHubClient) GetCommitMessage(ctx context.Context, repo string, sha string) (string, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return "", fmt.Errorf("malformed repo: %v", repo)
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	rc, _, err := ghc.c.Repositories.GetCommit(ctx, rs[0], rs[1], sha)
	if err != nil {
		return "", fmt.Errorf("error getting commit info: %v", err)
	}
	return *rc.Commit.Message, nil
}

// MaxFileDownloadSizeBytes contains the size limit for file content downloads. Attempting to GetFileContents for a file larger than this will return an error.
var MaxFileDownloadSizeBytes = 500 * 1000000

// GetFileContents returns the file contents of path (which must be a regular file or a symlink to a regular file) in the specified repo at ref
func (ghc *GitHubClient) GetFileContents(ctx context.Context, repo string, path string, ref string) ([]byte, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return nil, fmt.Errorf("malformed repo: %v", repo)
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	fc, _, _, err := ghc.c.Repositories.GetContents(ctx, rs[0], rs[1], path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, fmt.Errorf("error getting GitHub repo contents: %v", err)
	}
	if fc.GetSize() > MaxFileDownloadSizeBytes {
		return nil, fmt.Errorf("file size exceeds max (%v bytes): %v", MaxFileDownloadSizeBytes, fc.GetSize())
	}
	switch {
	case fc.GetType() == "file" || fc.GetType() == "symlink":
		if fc.GetTarget() != "" {
			return nil, fmt.Errorf("path is a symlink and target isn't a normal file: %v", fc.GetTarget())
		}
		if fc.GetSize() <= 1000*1024 {
			c, err := fc.GetContent()
			if err != nil {
				return nil, fmt.Errorf("error decoding GitHub repo contents: %v", err)
			}
			return []byte(c), nil
		}
		rc, err := ghc.c.Repositories.DownloadContents(ctx, rs[0], rs[1], path, &github.RepositoryContentGetOptions{Ref: ref})
		if err != nil {
			return nil, errors.Wrap(err, "error downloading file")
		}
		defer rc.Close()
		c, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, errors.Wrap(err, "error reading file contents")
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unsupported type for path: %v", fc.GetType())
	}
}

// FileContents models a file from a repository
type FileContents struct {
	Path, SymlinkTarget string
	Symlink             bool
	Contents            []byte
}

// GetDirectoryContents fetches path from repo at ref, returning a map of file path to file contents. If path is not a directory, an error is returned.
// The contents may include symlinks that point to arbitrary locations (ie, outside of the directory root).
func (ghc *GitHubClient) GetDirectoryContents(ctx context.Context, repo, path, ref string) (map[string]FileContents, error) {
	rs := strings.Split(repo, "/")
	if len(rs) != 2 {
		return nil, fmt.Errorf("malformed repo: %v", repo)
	}
	ctx, cf := context.WithTimeout(ctx, ghTimeout)
	defer cf()
	hc := http.Client{}
	// recursively fetch all directories and files
	var getDirContents func(dirpath string) (map[string]FileContents, error)
	getDirContents = func(dirpath string) (map[string]FileContents, error) {
		_, dc, _, err := ghc.c.Repositories.GetContents(ctx, rs[0], rs[1], dirpath, &github.RepositoryContentGetOptions{Ref: ref})
		if err != nil {
			return nil, errors.Wrap(err, "error getting GitHub repo contents")
		}
		if dc == nil {
			return nil, fmt.Errorf("directory contents is nil, path may not be a directory: %v", path)
		}
		output := map[string]FileContents{}
		getFile := func(fc *github.RepositoryContent) error {
			resp, err := hc.Get(fc.GetDownloadURL())
			if err != nil {
				return errors.Wrapf(err, "error downloading file: %v", fc.GetPath())
			}
			defer resp.Body.Close()
			c, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrapf(err, "error reading HTTP body for file: %v", fc.GetPath())
			}
			if len(c) != fc.GetSize() {
				return fmt.Errorf("unexpected size for %v: %v (wanted %v)", fc.GetPath(), len(c), fc.GetSize())
			}
			output[fc.GetPath()] = FileContents{
				Path:     fc.GetPath(),
				Contents: c,
			}
			return nil
		}
		for _, fc := range dc {
			switch fc.GetType() {
			case "file":
				if err := getFile(fc); err != nil {
					return nil, err
				}
			case "dir":
				out, err := getDirContents(fc.GetPath())
				if err != nil {
					return nil, errors.Wrapf(err, "error getting directory contents for %v", fc.GetPath())
				}
				for k, v := range out {
					output[k] = v
				}
			case "symlink":
				// if it's a symlink, we have to do another API call to get details
				fc2, _, _, err := ghc.c.Repositories.GetContents(ctx, rs[0], rs[1], fc.GetPath(), &github.RepositoryContentGetOptions{Ref: ref})
				if err != nil {
					return nil, errors.Wrap(err, "error getting symlink details")
				}
				if fc2.GetTarget() != "" {
					output[fc.GetPath()] = FileContents{
						Path:          fc2.GetPath(),
						SymlinkTarget: fc2.GetTarget(),
						Symlink:       true,
					}
					continue
				}
				if err := getFile(fc2); err != nil {
					return nil, err
				}
			default:
				// ignore any other types (submodule, etc)
				continue
			}
		}
		return output, nil
	}
	return getDirContents(path)
}
