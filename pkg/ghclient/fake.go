package ghclient

import "context"

// FakeRepoClient is fake implementaiton of RepoClient that runs user-supplied functions for each method
type FakeRepoClient struct {
	GetBranchFunc            func(context.Context, string, string) (BranchInfo, error)
	GetBranchesFunc          func(context.Context, string) ([]BranchInfo, error)
	GetTagsFunc              func(context.Context, string) ([]BranchInfo, error)
	SetStatusFunc            func(context.Context, string, string, *CommitStatus) error
	GetPRStatusFunc          func(context.Context, string, uint) (string, error)
	GetCommitMessageFunc     func(context.Context, string, string) (string, error)
	GetFileContentsFunc      func(ctx context.Context, repo string, path string, ref string) ([]byte, error)
	GetDirectoryContentsFunc func(ctx context.Context, repo, path, ref string) (map[string]FileContents, error)
}

var _ RepoClient = &FakeRepoClient{}

func (frc *FakeRepoClient) GetBranch(ctx context.Context, repo string, branch string) (BranchInfo, error) {
	if frc.GetBranchFunc != nil {
		return frc.GetBranchFunc(ctx, repo, branch)
	}
	return BranchInfo{}, nil
}
func (frc *FakeRepoClient) GetBranches(ctx context.Context, repo string) ([]BranchInfo, error) {
	if frc.GetBranchesFunc != nil {
		return frc.GetBranchesFunc(ctx, repo)
	}
	return []BranchInfo{}, nil
}
func (frc *FakeRepoClient) GetTags(ctx context.Context, repo string) ([]BranchInfo, error) {
	if frc.GetTagsFunc != nil {
		return frc.GetTagsFunc(ctx, repo)
	}
	return []BranchInfo{}, nil
}
func (frc *FakeRepoClient) SetStatus(ctx context.Context, repo string, sha string, status *CommitStatus) error {
	if frc.SetStatusFunc != nil {
		return frc.SetStatusFunc(ctx, repo, sha, status)
	}
	return nil
}
func (frc *FakeRepoClient) GetPRStatus(ctx context.Context, repo string, pr uint) (string, error) {
	if frc.GetPRStatusFunc != nil {
		return frc.GetPRStatusFunc(ctx, repo, pr)
	}
	return "", nil
}
func (frc *FakeRepoClient) GetCommitMessage(ctx context.Context, repo string, sha string) (string, error) {
	if frc.GetCommitMessageFunc != nil {
		return frc.GetCommitMessageFunc(ctx, repo, sha)
	}
	return "", nil
}
func (frc *FakeRepoClient) GetFileContents(ctx context.Context, repo string, path string, ref string) ([]byte, error) {
	if frc.GetFileContentsFunc != nil {
		return frc.GetFileContentsFunc(ctx, repo, path, ref)
	}
	return []byte{}, nil
}
func (frc *FakeRepoClient) GetDirectoryContents(ctx context.Context, repo, path, ref string) (map[string]FileContents, error) {
	if frc.GetDirectoryContentsFunc != nil {
		return frc.GetDirectoryContentsFunc(ctx, repo, path, ref)
	}
	return map[string]FileContents{}, nil
}
