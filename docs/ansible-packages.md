# Ansible package playbooks

Package lifecycle is driven by Ansible playbooks under `ansible/playbooks/packages/<id>/`.

## Layout

```text
ansible/
  ansible.cfg
  inventory/localhost.yml
  roles/common/
  playbooks/packages/<id>/
    install.yml
    upgrade.yml
    remove.yml
```

Installed on the host at `/usr/share/brrewery/ansible` by `scripts/install.sh`.

## MVP

Playbooks are **syntax-valid stubs** (local connection, placeholder `debug` task). CI runs:

```bash
make ansible-syntax-check
```

## M2 execution

The Go runner in `internal/packages/ansible` will invoke:

```bash
ansible-playbook --connection=local <playbook> -e @extra-vars.json
```

Extra-vars for package secrets are supplied from the API/UI per install and are **never** written to disk by brrewery.

After a successful run, install status is re-probed via filesystem detection only (no playbook marker files).
