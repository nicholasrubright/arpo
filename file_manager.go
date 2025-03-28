package main


import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func DirectoryExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }
    if errors.Is(err, fs.ErrNotExist) {
        return false, nil
    }
    return false, err
}

func MoveDirectories(src string, dest string) error {

	// check if dest directory exists
	dirExists, err := DirectoryExists(dest)
	if err != nil {
		return err
	}


	if !dirExists {
		/*
		err = os.Mkdir(dest, 0755)
		if err != nil {
			return err
		}
		*/
	}

	// move them
	/*
		err = os.Rename(src, dest)
		if err != nil {
			return err
		}
	*/

	return nil
}

func GetProjectDirectories() ([]project, error) {

	var projects []project

	entries, err := os.ReadDir(dev_path)

	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			p := project{
				name: entry.Name(),
				path: filepath.Join(dev_path, entry.Name()),
			}

			projects = append(projects, p)
		}
	}

	return projects, nil

}