			repo: makeGitRepository(t, gitCommands...),
							Author:    git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
							Committer: &git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
							Author:    git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
							Committer: &git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
							Author:    git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
							Committer: &git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
							Author:    git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
							Committer: &git.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				t.Errorf("%s: %+v: got %+v, want %+v", label, *opt, asJSON(results), asJSON(want))
			repo: makeGitRepository(t, gitCommands...),
				t.Errorf("%s: %+v: got %+v, want %+v", label, *opt, asJSON(results), asJSON(want))