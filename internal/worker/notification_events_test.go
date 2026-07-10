package worker

import "testing"

func TestNotificationLinksPointToApplicationTabs(t *testing.T) {
	runner := NewRunner(nil, Options{PublicBaseURL: "https://devops.example.com/"})

	links := runner.notificationLinks("prj_1", "app_1", "deployments", "release")

	if links["project"] != "https://devops.example.com/projects/prj_1" {
		t.Fatalf("project link = %q", links["project"])
	}
	if links["application"] != "https://devops.example.com/projects/prj_1/apps/app_1" {
		t.Fatalf("application link = %q", links["application"])
	}
	if links["release"] != "https://devops.example.com/projects/prj_1/apps/app_1#tab=deployments" {
		t.Fatalf("release link = %q", links["release"])
	}
	if links["primary"] != links["release"] {
		t.Fatalf("primary link = %q, release link = %q", links["primary"], links["release"])
	}
}

func TestNotificationLinksStayEmptyWithoutPublicBaseURL(t *testing.T) {
	runner := NewRunner(nil, Options{})

	if links := runner.notificationLinks("prj_1", "app_1", "builds", "build"); links != nil {
		t.Fatalf("links = %#v", links)
	}
}
