# Local Transcript Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a reusable `local-transcript` skill that transcribes a local video into a cleaned final `.txt` file with language-aware output.

**Architecture:** Scaffold a skill with the skill-creator tooling, implement one deterministic Python script that orchestrates `ffmpeg` and `whisper.cpp`, and define a concise skill contract that always outputs only the cleaned final transcript.

**Tech Stack:** Markdown skill files, Python 3, `ffmpeg`, `uvx`, `whisper.cpp-cli`

---

### Task 1: Create planning artifacts

**Files:**
- Create: `docs/plans/2026-03-07-local-transcript-design.md`
- Create: `docs/plans/2026-03-07-local-transcript.md`

**Step 1: Write the design doc**

Describe the approved workflow, dependencies, cleanup behavior, and testing scope.

**Step 2: Write the implementation plan**

Capture the scaffold, script, validation, smoke test, and installation steps.

### Task 2: Scaffold the skill

**Files:**
- Create: `.tmp-skills/local-transcript/**`

**Step 1: Run the skill initializer**

Run `init_skill.py` to create the skill skeleton with `scripts/`.

**Step 2: Verify generated files**

Confirm `SKILL.md`, `agents/openai.yaml`, and `scripts/` exist.

### Task 3: Implement the skill

**Files:**
- Modify: `.tmp-skills/local-transcript/SKILL.md`
- Create: `.tmp-skills/local-transcript/scripts/local_transcript.py`

**Step 1: Write the deterministic script**

Handle input validation, model bootstrap, audio extraction, transcription, cleanup, and output writing.

**Step 2: Write the skill contract**

Describe when to use the skill, required checks, defaults, and the final output contract.

### Task 4: Validate and test

**Files:**
- Modify: `.tmp-skills/local-transcript/**` as needed

**Step 1: Validate skill structure**

Run `quick_validate.py` against the skill folder.

**Step 2: Run a smoke test**

Execute the script on the sample video in `~/Downloads/yt-dlp`.

**Step 3: Inspect transcript output**

Confirm the final `.txt` is non-empty, paragraphized, and language-appropriate.

### Task 5: Install globally

**Files:**
- Create: `~/.codex/skills/local-transcript/**`

**Step 1: Copy validated skill to the global skills directory**

Install the finished skill under `~/.codex/skills/local-transcript`.

**Step 2: Verify installed files**

List the target directory contents and ensure the expected files exist.
