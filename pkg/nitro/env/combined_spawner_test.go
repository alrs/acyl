package env

import (
	"context"
	"testing"

	"github.com/dollarshaveclub/acyl/pkg/models"
	"github.com/dollarshaveclub/acyl/pkg/nitro/meta"
	"github.com/dollarshaveclub/acyl/pkg/persistence"
	"github.com/dollarshaveclub/acyl/pkg/spawner"
)

func TestCombinedSpawnerExtantUsedNitro(t *testing.T) {
	env1 := &models.KubernetesEnvironment{EnvName: "foo-bar"}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), &models.QAEnvironment{Name: env1.EnvName})
	dl.CreateK8sEnv(context.Background(), env1)
	cb := CombinedSpawner{DL: dl}
	ok, err := cb.extantUsedNitro(context.Background(), env1.EnvName)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if !ok {
		t.Fatalf("expected true but got false")
	}
	ok, err = cb.extantUsedNitro(context.Background(), "something-else")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ok {
		t.Fatalf("expected false but got true")
	}
}

func TestCombinedSpawnerExtantUsedNitroFromRDD(t *testing.T) {
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	env2 := &models.QAEnvironment{Name: "foo-bar2", Repo: "foo/bar2", PullRequest: 1, AminoEnvironmentID: 23}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	dl.CreateQAEnvironment(context.Background(), env2)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
	cb := CombinedSpawner{DL: dl}
	ok, err := cb.extantUsedNitroFromRDD(context.Background(), *env1.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if !ok {
		t.Fatalf("expected true but got false")
	}
	ok, err = cb.extantUsedNitroFromRDD(context.Background(), *env2.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ok {
		t.Fatalf("expected false but got true")
	}
}

func TestCombinedSpawnerIsAcylYAMLV2(t *testing.T) {
	var v2 bool
	mg := &meta.FakeGetter{
		GetAcylYAMLFunc: func(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error) {
			if v2 {
				return nil
			}
			return meta.ErrUnsupportedVersion
		},
	}
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	cb := CombinedSpawner{MG: mg}
	ok, err := cb.isAcylYMLV2(context.Background(), *env1.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ok {
		t.Fatalf("expected false but got true")
	}
	v2 = true
	ok, err = cb.isAcylYMLV2(context.Background(), *env1.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if !ok {
		t.Fatalf("expected true but got false")
	}
}

func TestCombinedSpawnerIsAcylYAMLV2FromName(t *testing.T) {
	var v2 bool
	mg := &meta.FakeGetter{
		GetAcylYAMLFunc: func(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error) {
			if v2 {
				return nil
			}
			return meta.ErrUnsupportedVersion
		},
	}
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	cb := CombinedSpawner{MG: mg, DL: dl}
	ok, err := cb.isAcylYMLV2FromName(context.Background(), env1.Name)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ok {
		t.Fatalf("expected false but got true")
	}
	v2 = true
	ok, err = cb.isAcylYMLV2FromName(context.Background(), env1.Name)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if !ok {
		t.Fatalf("expected true but got false")
	}
}

func TestCombinedSpawnerCreate(t *testing.T) {
	var v2 bool
	mg := &meta.FakeGetter{
		GetAcylYAMLFunc: func(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error) {
			if v2 {
				return nil
			}
			return meta.ErrUnsupportedVersion
		},
	}
	var nitrocalled, aminocalled bool
	fknitro := &spawner.FakeEnvironmentSpawner{
		CreateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
			nitrocalled = true
			return "foo-bar", nil
		},
	}
	fkamino := &spawner.FakeEnvironmentSpawner{
		CreateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
			aminocalled = true
			return "foo-bar", nil
		},
	}
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	cb := CombinedSpawner{MG: mg, Nitro: fknitro, Spawner: fkamino}
	_, err := cb.Create(context.Background(), *env1.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if nitrocalled || !aminocalled {
		t.Fatalf("expected amino and not nitro")
	}
	v2 = true
	nitrocalled, aminocalled = false, false
	_, err = cb.Create(context.Background(), *env1.RepoRevisionDataFromQA())
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if aminocalled || !nitrocalled {
		t.Fatalf("expected nitro and not amino")
	}
}

func TestCombinedSpawnerDestroy(t *testing.T) {
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	env2 := &models.QAEnvironment{Name: "foo-bar2", Repo: "foo/bar2", PullRequest: 1, AminoEnvironmentID: 23}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
	dl.CreateQAEnvironment(context.Background(), env2)
	var nitrocalled, aminocalled bool
	fknitro := &spawner.FakeEnvironmentSpawner{
		DestroyFunc: func(ctx context.Context, rd models.RepoRevisionData, reason models.QADestroyReason) error {
			nitrocalled = true
			return nil
		},
	}
	fkamino := &spawner.FakeEnvironmentSpawner{
		DestroyFunc: func(ctx context.Context, rd models.RepoRevisionData, reason models.QADestroyReason) error {
			aminocalled = true
			return nil
		},
	}
	cb := CombinedSpawner{DL: dl, Nitro: fknitro, Spawner: fkamino}
	if err := cb.Destroy(context.Background(), *env1.RepoRevisionDataFromQA(), models.DestroyApiRequest); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if aminocalled || !nitrocalled {
		t.Fatalf("expected nitro and not amino")
	}
	nitrocalled, aminocalled = false, false
	if err := cb.Destroy(context.Background(), *env2.RepoRevisionDataFromQA(), models.DestroyApiRequest); err != nil {
		t.Fatalf("should have succeeded 2: %v", err)
	}
	if nitrocalled || !aminocalled {
		t.Fatalf("expected amino and not nitro")
	}
}

func TestCombinedSpawnerDestroyExplicitly(t *testing.T) {
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	env2 := &models.QAEnvironment{Name: "foo-bar2", Repo: "foo/bar2", PullRequest: 1, AminoEnvironmentID: 23}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
	dl.CreateQAEnvironment(context.Background(), env2)
	var nitrocalled, aminocalled bool
	fknitro := &spawner.FakeEnvironmentSpawner{
		DestroyExplicitlyFunc: func(ctx context.Context, env *models.QAEnvironment, reason models.QADestroyReason) error {
			nitrocalled = true
			return nil
		},
	}
	fkamino := &spawner.FakeEnvironmentSpawner{
		DestroyExplicitlyFunc: func(ctx context.Context, env *models.QAEnvironment, reason models.QADestroyReason) error {
			aminocalled = true
			return nil
		},
	}
	cb := CombinedSpawner{DL: dl, Nitro: fknitro, Spawner: fkamino}
	if err := cb.DestroyExplicitly(context.Background(), env1, models.DestroyApiRequest); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if aminocalled || !nitrocalled {
		t.Fatalf("expected nitro and not amino")
	}
	nitrocalled, aminocalled = false, false
	if err := cb.DestroyExplicitly(context.Background(), env2, models.DestroyApiRequest); err != nil {
		t.Fatalf("should have succeeded 2: %v", err)
	}
	if nitrocalled || !aminocalled {
		t.Fatalf("expected amino and not nitro")
	}
}

func TestCombinedSpawnerSuccess(t *testing.T) {
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	env2 := &models.QAEnvironment{Name: "foo-bar2", Repo: "foo/bar2", PullRequest: 1, AminoEnvironmentID: 23}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
	dl.CreateQAEnvironment(context.Background(), env2)
	var nitrocalled, aminocalled bool
	fknitro := &spawner.FakeEnvironmentSpawner{
		SuccessFunc: func(ctx context.Context, name string) error {
			nitrocalled = true
			return nil
		},
	}
	fkamino := &spawner.FakeEnvironmentSpawner{
		SuccessFunc: func(ctx context.Context, name string) error {
			aminocalled = true
			return nil
		},
	}
	cb := CombinedSpawner{DL: dl, Nitro: fknitro, Spawner: fkamino}
	if err := cb.Success(context.Background(), env1.Name); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if aminocalled || !nitrocalled {
		t.Fatalf("expected nitro and not amino")
	}
	nitrocalled, aminocalled = false, false
	if err := cb.Success(context.Background(), env2.Name); err != nil {
		t.Fatalf("should have succeeded 2: %v", err)
	}
	if nitrocalled || !aminocalled {
		t.Fatalf("expected amino and not nitro")
	}
}

func TestCombinedSpawnerFailure(t *testing.T) {
	env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
	env2 := &models.QAEnvironment{Name: "foo-bar2", Repo: "foo/bar2", PullRequest: 1, AminoEnvironmentID: 23}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env1)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
	dl.CreateQAEnvironment(context.Background(), env2)
	var nitrocalled, aminocalled bool
	fknitro := &spawner.FakeEnvironmentSpawner{
		FailureFunc: func(ctx context.Context, name, msg string) error {
			nitrocalled = true
			return nil
		},
	}
	fkamino := &spawner.FakeEnvironmentSpawner{
		FailureFunc: func(ctx context.Context, name, msg string) error {
			aminocalled = true
			return nil
		},
	}
	cb := CombinedSpawner{DL: dl, Nitro: fknitro, Spawner: fkamino}
	if err := cb.Failure(context.Background(), env1.Name, ""); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if aminocalled || !nitrocalled {
		t.Fatalf("expected nitro and not amino")
	}
	nitrocalled, aminocalled = false, false
	if err := cb.Failure(context.Background(), env2.Name, ""); err != nil {
		t.Fatalf("should have succeeded 2: %v", err)
	}
	if nitrocalled || !aminocalled {
		t.Fatalf("expected amino and not nitro")
	}
}

func TestCombinedSpawnerUpdate(t *testing.T) {
	cases := []struct {
		name                                                                                              string
		v2, nitro, nitroupdated, nitrocreated, nitrodestroyed, aminoupdated, aminocreated, aminodestroyed bool
	}{
		{
			"acyl v2, nitro env", true, true, true, false, false, false, false, false,
		},
		{
			"acyl v2, amino env", true, false, false, true, false, false, false, true,
		},
		{
			"acyl v1, nitro env", false, true, false, false, true, false, true, false,
		},
		{
			"acyl v1, amino env", false, false, false, false, false, true, false, false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mg := &meta.FakeGetter{
				GetAcylYAMLFunc: func(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error) {
					if c.v2 {
						return nil
					}
					return meta.ErrUnsupportedVersion
				},
			}
			env1 := &models.QAEnvironment{Name: "foo-bar", Repo: "foo/bar", PullRequest: 1}
			if !c.nitro {
				env1.AminoEnvironmentID = 23
			}
			dl := persistence.NewFakeDataLayer()
			dl.CreateQAEnvironment(context.Background(), env1)
			if c.nitro {
				dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{EnvName: env1.Name})
			}
			var nitrodestroycalled, aminodestroycalled bool
			var nitrocreatecalled, aminocreatecalled bool
			var nitroupdatecalled, aminoupdatecalled bool
			fknitro := &spawner.FakeEnvironmentSpawner{
				CreateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
					nitrocreatecalled = true
					return "foo-bar", nil
				},
				UpdateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
					nitroupdatecalled = true
					return "foo-bar", nil
				},
				DestroyFunc: func(ctx context.Context, rd models.RepoRevisionData, reason models.QADestroyReason) error {
					nitrodestroycalled = true
					return nil
				},
			}
			fkamino := &spawner.FakeEnvironmentSpawner{
				CreateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
					aminocreatecalled = true
					return "foo-bar", nil
				},
				UpdateFunc: func(ctx context.Context, rd models.RepoRevisionData) (string, error) {
					aminoupdatecalled = true
					return "foo-bar", nil
				},
				DestroyFunc: func(ctx context.Context, rd models.RepoRevisionData, reason models.QADestroyReason) error {
					aminodestroycalled = true
					return nil
				},
			}
			cb := CombinedSpawner{MG: mg, DL: dl, Nitro: fknitro, Spawner: fkamino}
			if _, err := cb.Update(context.Background(), *env1.RepoRevisionDataFromQA()); err != nil {
				t.Fatalf("should have succeeded: %v", err)
			}
			if c.nitrocreated != nitrocreatecalled {
				t.Fatalf("nitro created: %v; called: %v", c.nitrocreated, nitrocreatecalled)
			}
			if c.nitroupdated != nitroupdatecalled {
				t.Fatalf("nitro updated: %v; called: %v", c.nitroupdated, nitroupdatecalled)
			}
			if c.nitrodestroyed != nitrodestroycalled {
				t.Fatalf("nitro destroyed: %v; called: %v", c.nitrodestroyed, nitrodestroycalled)
			}
			if c.aminocreated != aminocreatecalled {
				t.Fatalf("amino created: %v; called: %v", c.aminocreated, aminocreatecalled)
			}
			if c.aminoupdated != aminoupdatecalled {
				t.Fatalf("amino updated: %v; called: %v", c.aminoupdated, aminoupdatecalled)
			}
			if c.aminodestroyed != aminodestroycalled {
				t.Fatalf("amino destroyed: %v; called: %v", c.aminodestroyed, aminodestroycalled)
			}
		})
	}
}
