<h1 align="center">
  nucleihub
  <br>
</h1>
<h4 align="center">A simple tool that allows you to organize all the Nuclei templates offered by the community in one place, Inspired by https://github.com/xm1k3/cent.</h4>

### nucleihub supports all of these url types, in the end must contain `.git, .yaml, .zip`
```
https://github.com/rix4uni/nucleihub.git
https://gist.githubusercontent.com/0x240x23elu/a450c1829de9bb4559ea0243bcc70714/raw/edcb9e91b6c1b1fa842b14e391c20bb1e2ef4c81/CVE-2023-26255.yaml
https://github.com/projectdiscovery/nuclei-templates/archive/refs/heads/main.zip
https://raw.githubusercontent.com/boobooHQ/private_templates/refs/heads/main/springboot_heapdump.yaml
```

## Installation
```
go install github.com/rix4uni/nucleihub@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/nucleihub/releases/download/v0.0.3/nucleihub-linux-amd64-0.0.3.tgz
tar -xvzf nucleihub-linux-amd64-0.0.3.tgz
rm -rf nucleihub-linux-amd64-0.0.3.tgz
mv nucleihub ~/go/bin/nucleihub
```
Or download [binary release](https://github.com/rix4uni/nucleihub/releases) for your platform.

## Compile from source
```
git clone --depth 1 github.com/rix4uni/nucleihub.git
cd nucleihub; go install
```

## Usage
```console
nucleihub -h

                        __       _  __            __
   ____   __  __ _____ / /___   (_)/ /_   __  __ / /_
  / __ \ / / / // ___// // _ \ / // __ \ / / / // __ \
 / / / // /_/ // /__ / //  __// // / / // /_/ // /_/ /
/_/ /_/ \__,_/ \___//_/ \___//_//_/ /_/ \__,_//_.___/

                            Current nucleihub version v0.0.3

Community edition nuclei templates, a simple tool that allows you
to organize all the Nuclei templates offered by the community in one place.

Examples:
  # Step 1, download
  cat reponames.txt | nucleihub download --output-directory ~/nucleihub-downloaded-repos

  # Step 2, remove duplicates
  nucleihub duplicate --input-directory ~/nucleihub-downloaded-repos --output-directory ~/nucleihub-templates --large-content

Usage:
  nucleihub [flags]
  nucleihub [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  download    Clone or download repositories and files with various options
  duplicate   Find and save unique or large content templates from downloaded files
  help        Help about any command
  updatecheck Check for today's commits in specified GitHub repositories

Flags:
  -h, --help      help for nucleihub
  -u, --update    update nucleihub to latest version
  -v, --version   Print the version of the tool and exit.

Use "nucleihub [command] --help" for more information about a command.
```