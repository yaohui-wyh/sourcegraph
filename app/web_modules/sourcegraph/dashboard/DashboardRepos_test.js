import autotest from "sourcegraph/util/autotest";

import React from "react";
import moment from "moment";
import DashboardRepos from "sourcegraph/dashboard/DashboardRepos";

import testdataData from "sourcegraph/dashboard/testdata/DashboardRepos-data.json";
import testdataNotSupported from "sourcegraph/dashboard/testdata/DashboardRepos-notSupported.json";
import testdataOnWaitlist from "sourcegraph/dashboard/testdata/DashboardRepos-onWaitlist.json";

describe("DashboardRepos", () => {
	it("should render repos", () => {
		autotest(testdataData, `${__dirname}/testdata/DashboardRepos-data.json`,
			<DashboardRepos
				repos={[{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "Go"}]}
				onWaitlist={false}
				allowGitHubMirrors={true}
				linkGitHub={false} />
		);
	});
});

describe("DashboardRepos", () => {
	it("should render repos", () => {
		let repos = [
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "C++"},
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "C"},
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "Python"}];
		autotest(testdataNotSupported, `${__dirname}/testdata/DashboardRepos-notSupported.json`,
			<DashboardRepos
				repos={repos}
				onWaitlist={false}
				allowGitHubMirrors={true}
				linkGitHub={false} />
		);
	});
});

describe("DashboardRepos", () => {
	it("should render repos on waitlist", () => {
		let repos = [
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "C++"},
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "C"},
			{Private: false, URI: "someURL", Description: "someDescription", UpdatedAt: moment(), Language: "Python"}];
		autotest(testdataOnWaitlist, `${__dirname}/testdata/DashboardRepos-onWaitlist.json`,
			<DashboardRepos
				repos={repos}
				onWaitlist={true}
				allowGitHubMirrors={true}
				linkGitHub={false} />
		);
	});
});
