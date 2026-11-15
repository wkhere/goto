* Template tests
* migrate to github; use github actions
* use own sizefmt
* rename agent to actor; use act/ dir
* fix printing handlers in dry-run

### v0.7.1
* use lowercase "ok" in messages
* doc tweaks

### v0.7.0
* migrate to codeberg

### v0.6.4
* runner args: unfold one-letter flags

### v0.6.3
* Copy, Template: respect file modes fully

### v0.6.2
* runner: load .env

### v0.6.1
* Template: descriptive error msg for the missing file mode

### v0.6.0
* core: new eval with cross-vars resolution & subvars, using topological sort

### v0.5.4
* Copy: bindiff: fix the order of errors

### v0.5.3
* Copy: optimize rsync stats parsing

### v0.5.2
* indicate dry-run & empty tasks

### v0.5.1
* runner: fix tags passing (bug occured for no tags)
* print program name on error exit for both runner & agent

### v0.5.0
* agent phases for remote/local execution
* Copy task
* the runner

Detailed notes:

* tmpPath is now without pid part
* Copy prep: uses hard link instead of file copying <br>
  It relies on the rsync implementation: rsync should disconnect
  the link if files differ during transfer.
  Saves I/O when the files don't differ.
* Copy: in diff, show actual real names of the files (no artificial -new suffix)
* same goes for Transfer

### v0.4.2
* Template: compile-time interface check

### v0.4.1
* DedupHandlers: in-place delete

### v0.4.0
* DedupHandlers
* Template.Mode
* fix Template logic for new files

### v0.3.0
* new Template API

### v0.2.1
* formatting fix

### v0.2.0
* notify
* handlers
* internal api tweaks (evalTo, readFile)

### v0.1.0
* core API & Template task - preliminary version
