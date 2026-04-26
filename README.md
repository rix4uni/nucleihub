<h1 align="center">
  nucleihub
  <br>
</h1>
<h4 align="center">A simple tool that allows you to organize all the Nuclei templates offered by the community in one place, Inspired by https://github.com/xm1k3/cent.</h4>

<img width="1690" height="992" alt="image" src="https://github.com/user-attachments/assets/dbfc26b9-16cd-4ea6-ab9b-1bc09c761e48" />


### nucleihub supports all of these url types (must end with `.git`, `.yaml`, or `.zip`)
```console
https://github.com/rix4uni/nucleihub.git
https://gist.githubusercontent.com/0x240x23elu/a450c1829de9bb4559ea0243bcc70714/raw/edcb9e91b6c1b1fa842b14e391c20bb1e2ef4c81/CVE-2023-26255.yaml
https://github.com/projectdiscovery/nuclei-templates/archive/refs/heads/main.zip
https://raw.githubusercontent.com/boobooHQ/private_templates/refs/heads/main/springboot_heapdump.yaml
```

### Features
- **Auto-flatten**: Moves all `.yaml` files from subdirectories to the root output directory
- **Duplicate handling**: When duplicate filenames exist, keeps the larger file
- **Smart filtering**: Automatically removes hash-suffixed (e.g., `file-d41d8cd98f00b204e9800998ecf8427e.yaml`) and numeric-suffixed (e.g., `file-23.yaml`) duplicates
- **Nuclei validation**: Validates all templates after download (skippable with `--no-validate`)
- **Low resource usage**: Optimized for 1GB VPS environments with buffered I/O and controlled concurrency

## Installation
```
go install github.com/rix4uni/nucleihub@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/nucleihub/releases/download/v0.0.4/nucleihub-linux-amd64-0.0.4.tgz
tar -xvzf nucleihub-linux-amd64-0.0.4.tgz
rm -rf nucleihub-linux-amd64-0.0.4.tgz
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
Usage of nucleihub:
  -o, --output-directory string   Directory to download into (default "~/nucleihub-templates")
  -p, --parallel int              Number of operations to perform in parallel (default 10)
      --no-validate               Skip post-download nuclei validation
      --silent                    Silent mode (no banner)
      --version                   Print version and exit
```

### Usage Examples
**Basic download (read URLs from stdin):**
```console
cat reponames.txt | nucleihub
```

**Specify custom output directory:**
```console
cat reponames.txt | nucleihub -o ~/my-templates
```

**Reduce parallel operations for low-resource VPS:**
```console
cat reponames.txt | nucleihub -p 5
```

**Skip nuclei validation (faster):**
```console
cat reponames.txt | nucleihub --no-validate
```
