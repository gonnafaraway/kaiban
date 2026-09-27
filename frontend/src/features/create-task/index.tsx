"use client";

import { useState } from "react";
import { api } from "@/shared/api";
import { Button, Input, Textarea } from "@/shared/ui";
import { t } from "@/shared/i18n";

export function CreateTask({ locale, onCreated }: { locale: string; onCreated: () => void }) {
  const d = t(locale);
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [jira, setJira] = useState("");
  const [confluence, setConfluence] = useState("");
  const [gitlab, setGitlab] = useState("");
  const [github, setGithub] = useState("");
  return (
    <>
      <Button onClick={() => setOpen(true)}>{d.createTask}</Button>
      {open && (
        <div className="modal">
          <div className="modal-card">
            <h3>{d.createTask}</h3>
            <label>{d.title}</label>
            <Input placeholder={d.title} value={title} onChange={(e) => setTitle(e.target.value)} />
            <label>{d.description}</label>
            <Textarea placeholder={d.description} value={description} onChange={(e) => setDescription(e.target.value)} rows={5} />
            <h4>{d.artifacts}</h4>
            <label>{d.jiraIssue}</label>
            <Input placeholder={d.jiraPlaceholder} value={jira} onChange={(e) => setJira(e.target.value)} />
            <label>{d.confluenceUrl}</label>
            <Input placeholder="https://confluence.../pages/..." value={confluence} onChange={(e) => setConfluence(e.target.value)} />
            <label>{d.gitlabRepo}</label>
            <Input placeholder={d.gitlabPlaceholder} value={gitlab} onChange={(e) => setGitlab(e.target.value)} />
            <label>{d.githubRepo}</label>
            <Input placeholder={d.githubPlaceholder} value={github} onChange={(e) => setGithub(e.target.value)} />
            <div className="row">
              <Button
                onClick={async () => {
                  await api.createTask({
                    title,
                    description,
                    variables: {},
                    artifacts: {
                      jira_issue: jira.trim(),
                      confluence_url: confluence.trim(),
                      gitlab_repo: gitlab.trim(),
                      github_repo: github.trim(),
                    },
                  });
                  setOpen(false);
                  setTitle("");
                  setDescription("");
                  setJira("");
                  setConfluence("");
                  setGitlab("");
                  setGithub("");
                  onCreated();
                }}
              >
                {d.save}
              </Button>
              <Button type="button" className="btn--secondary" onClick={() => setOpen(false)}>
                ×
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
