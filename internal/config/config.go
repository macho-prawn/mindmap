package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Orgs []Org
}

type Org struct {
	Name      string
	Workloads []Workload
}

type Workload struct {
	Name string
	Envs []Environment
}

type Environment struct {
	Name      string
	ProjectID string
}

func Parse(data []byte) (Config, error) {
	lines := strings.Split(string(data), "\n")
	var cfg Config

	var currentOrg *Org
	var currentWorkload *Workload
	var currentEnv *Environment
	section := ""

	for idx, raw := range lines {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		switch {
		case indent == 0 && trimmed == "org:":
			section = "org-root"
		case indent == 2 && strings.HasPrefix(trimmed, "- name:"):
			if section != "org-root" && section != "org-item" && section != "workload-item" && section != "env-item" {
				return Config{}, lineError(idx, "unexpected org entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "org name cannot be empty")
			}
			cfg.Orgs = append(cfg.Orgs, Org{Name: name})
			currentOrg = &cfg.Orgs[len(cfg.Orgs)-1]
			currentWorkload = nil
			currentEnv = nil
			section = "org-item"
		case indent == 4 && trimmed == "workload:":
			if currentOrg == nil {
				return Config{}, lineError(idx, "workload section without org")
			}
			section = "workload-root"
		case indent == 6 && strings.HasPrefix(trimmed, "- name:"):
			if currentOrg == nil {
				return Config{}, lineError(idx, "workload entry without org")
			}
			if section != "workload-root" && section != "workload-item" && section != "env-item" {
				return Config{}, lineError(idx, "unexpected workload entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "workload name cannot be empty")
			}
			currentOrg.Workloads = append(currentOrg.Workloads, Workload{Name: name})
			currentWorkload = &currentOrg.Workloads[len(currentOrg.Workloads)-1]
			currentEnv = nil
			section = "workload-item"
		case indent == 8 && trimmed == "env:":
			if currentWorkload == nil {
				return Config{}, lineError(idx, "env section without workload")
			}
			section = "env-root"
		case indent == 10 && strings.HasPrefix(trimmed, "- name:"):
			if currentWorkload == nil {
				return Config{}, lineError(idx, "env entry without workload")
			}
			if section != "env-root" && section != "env-item" {
				return Config{}, lineError(idx, "unexpected env entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "env name cannot be empty")
			}
			currentWorkload.Envs = append(currentWorkload.Envs, Environment{Name: name})
			currentEnv = &currentWorkload.Envs[len(currentWorkload.Envs)-1]
			section = "env-item"
		case indent == 12 && strings.HasPrefix(trimmed, "project_id:"):
			if currentEnv == nil {
				return Config{}, lineError(idx, "project_id without env")
			}
			projectID := valueAfter(trimmed, "project_id:")
			if projectID == "" {
				return Config{}, lineError(idx, "project_id cannot be empty")
			}
			currentEnv.ProjectID = projectID
		default:
			return Config{}, lineError(idx, fmt.Sprintf("unsupported line %q", strings.TrimSpace(raw)))
		}
	}

	if len(cfg.Orgs) == 0 {
		return Config{}, fmt.Errorf("config does not define any org entries")
	}

	return cfg, nil
}

func (c Config) Resolve(orgName, workloadName, envName string) (string, error) {
	projects, err := c.ResolveProjects(orgName, workloadName, envName)
	if err != nil {
		return "", err
	}
	if len(projects) != 1 {
		return "", fmt.Errorf("expected exactly one project for org %q workload %q env %q, got %d", orgName, workloadName, envName, len(projects))
	}
	return projects[0], nil
}

func (c Config) ResolveProjects(orgName, workloadName, envName string) ([]string, error) {
	org, err := c.findOrg(orgName)
	if err != nil {
		return nil, err
	}

	workloads, err := findWorkloads(org, workloadName)
	if err != nil {
		return nil, err
	}

	var projects []string
	seen := make(map[string]struct{})
	if strings.TrimSpace(workloadName) == "" && strings.TrimSpace(envName) != "" {
		envFound := false
		for _, workload := range workloads {
			envs := matchingEnvironments(workload, envName)
			if len(envs) == 0 {
				continue
			}
			envFound = true
			for _, env := range envs {
				if strings.TrimSpace(env.ProjectID) == "" {
					return nil, fmt.Errorf("project_id is empty for org %q workload %q env %q", org.Name, workload.Name, env.Name)
				}
				if _, ok := seen[env.ProjectID]; ok {
					continue
				}
				seen[env.ProjectID] = struct{}{}
				projects = append(projects, env.ProjectID)
			}
		}
		if !envFound {
			return nil, fmt.Errorf("env %q not found under org %q", envName, org.Name)
		}
	} else {
		for _, workload := range workloads {
			envs, err := findEnvironments(org.Name, workload, envName)
			if err != nil {
				return nil, err
			}
			for _, env := range envs {
				if strings.TrimSpace(env.ProjectID) == "" {
					return nil, fmt.Errorf("project_id is empty for org %q workload %q env %q", org.Name, workload.Name, env.Name)
				}
				if _, ok := seen[env.ProjectID]; ok {
					continue
				}
				seen[env.ProjectID] = struct{}{}
				projects = append(projects, env.ProjectID)
			}
		}
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects matched org %q workload %q env %q", orgName, workloadName, envName)
	}
	return projects, nil
}

func matchingEnvironments(workload Workload, envName string) []Environment {
	var matches []Environment
	for _, env := range workload.Envs {
		if env.Name == envName {
			matches = append(matches, env)
		}
	}
	return matches
}

func (c Config) findOrg(orgName string) (Org, error) {
	var orgMatches []Org
	for _, org := range c.Orgs {
		if org.Name == orgName {
			orgMatches = append(orgMatches, org)
		}
	}
	if len(orgMatches) == 0 {
		return Org{}, fmt.Errorf("org %q not found in config", orgName)
	}
	if len(orgMatches) > 1 {
		return Org{}, fmt.Errorf("org %q is duplicated in config", orgName)
	}
	return orgMatches[0], nil
}

func findWorkloads(org Org, workloadName string) ([]Workload, error) {
	if strings.TrimSpace(workloadName) == "" {
		return org.Workloads, nil
	}

	var workloadMatches []Workload
	for _, workload := range org.Workloads {
		if workload.Name == workloadName {
			workloadMatches = append(workloadMatches, workload)
		}
	}
	if len(workloadMatches) == 0 {
		return nil, fmt.Errorf("workload %q not found under org %q", workloadName, org.Name)
	}
	if len(workloadMatches) > 1 {
		return nil, fmt.Errorf("workload %q is duplicated under org %q", workloadName, org.Name)
	}
	return workloadMatches, nil
}

func findEnvironments(orgName string, workload Workload, envName string) ([]Environment, error) {
	if strings.TrimSpace(envName) == "" {
		return workload.Envs, nil
	}

	var envMatches []Environment
	for _, env := range workload.Envs {
		if env.Name == envName {
			envMatches = append(envMatches, env)
		}
	}
	if len(envMatches) == 0 {
		return nil, fmt.Errorf("env %q not found under org %q workload %q", envName, orgName, workload.Name)
	}
	if len(envMatches) > 1 {
		return nil, fmt.Errorf("env %q is duplicated under org %q workload %q", envName, orgName, workload.Name)
	}
	return envMatches, nil
}

func stripComment(line string) string {
	var b strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '#':
			if !inQuote {
				return b.String()
			}
		case r == '"':
			inQuote = !inQuote
		}
		b.WriteRune(r)
	}
	return b.String()
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func valueAfter(line, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, prefix))
}

func lineError(index int, message string) error {
	return fmt.Errorf("config parse error on line %d: %s", index+1, message)
}
