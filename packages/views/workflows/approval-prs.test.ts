import { describe, expect, it } from "vitest";
import { collectPrs } from "./approval-prs";

describe("collectPrs", () => {
  it("lists linked PRs, then PR links from text, newest mention first, once each", () => {
    const linked = [{ html_url: "https://github.com/vantt/momentry/pull/3", repo_owner: "vantt", repo_name: "momentry", number: 3, title: "goals", state: "open" }];
    const texts = [
      "spec in https://github.com/vantt/momentry/pull/3",
      "Approve PR https://github.com/vantt/opc-studio-app/pull/8 on GitHub",
      "new round: https://github.com/vantt/opc-studio-app/pull/9",
    ];
    expect(collectPrs(linked, texts).map((p) => `${p.repo}#${p.number}`)).toEqual([
      "vantt/momentry#3",
      "vantt/opc-studio-app#9",
      "vantt/opc-studio-app#8",
    ]);
  });
  it("is empty without PRs", () => {
    expect(collectPrs([], ["no link here"])).toEqual([]);
  });
});
