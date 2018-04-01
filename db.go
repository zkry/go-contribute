package main

import (
	"database/sql"
)

func insertLabel(db *sql.DB, repo, label, color string, ct int) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO label(repo_name, name, ct, color) values (?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(repo, label, ct, color)
	if err != nil {
		return err
	}
	return nil
}

func insertRepo(db *sql.DB, repo string, starCt, forkCt int, desc string) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO repo(name, star_ct, fork_ct, description) values (?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(repo, starCt, forkCt, desc)
	if err != nil {
		return err
	}
	return nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE repo (name text primary key not null, star_ct int not null, fork_ct int not null, description text)")
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE label (repo_name TEXT NOT NULL, name TEXT NOT NULL, ct INT NOT NULL, color text, primary key (repo_name, name))")
	if err != nil {
		return err
	}
	return nil
}

func getHelpPageData(db *sql.DB) (HelpPageData, error) {

	query := `
	SELECT repo.name, label.name AS label_name, label.ct AS label_ct, label.color,
		   repo.star_ct, repo.fork_ct, repo.description, help_issue_ct FROM repo
	JOIN label ON repo.name=label.repo_name 
	JOIN (
		SELECT SUM(label.ct) AS help_issue_ct, repo.name AS lbct_name FROM repo 
		JOIN label ON repo.name=label.repo_name GROUP BY label.repo_name
	) ON lbct_name=repo.name
	ORDER BY help_issue_ct DESC, repo.star_ct DESC, repo.fork_ct DESC, repo.name`

	rows, err := db.Query(query)
	if err != nil {
		return HelpPageData{}, err
	}

	repos := []repositoryData{}

	var repoName string
	var labelName string
	var labelCt int
	var labelColor string
	var starCt int
	var forkCt int
	var description string
	var helpIssueCt int
	for rows.Next() {
		err := rows.Scan(&repoName, &labelName, &labelCt, &labelColor, &starCt, &forkCt, &description, &helpIssueCt)
		if err != nil {
			return HelpPageData{}, err
		}
		l := labelData{
			LabelName:     labelName,
			LabelCt:       labelCt,
			LabelColor:    labelColor,
			LabelTxtColor: colorFromBGColor(labelColor),
		}
		if len(repos) > 0 && repos[len(repos)-1].Name == repoName {
			// We are just adding a label
			repos[len(repos)-1].Labels = append(repos[len(repos)-1].Labels, l)
			continue
		}
		r := repositoryData{
			Name:        repoName,
			StarCt:      starCt,
			ForkCt:      forkCt,
			Description: description,
			Labels:      []labelData{l},
		}
		repos = append(repos, r)
	}

	data := HelpPageData{repos}
	return data, nil
}
