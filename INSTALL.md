# Installation

`rod-cli` is a compiled Go binary, which means it requires zero Node.js or Python dependencies.

## Option 1: Install via Go (Recommended)

If you have Go 1.23+ installed, you can install it globally:

```bash
go install github.com/agenthands/rod-cli@latest
```

This will place the `rod-cli` binary in your `$GOPATH/bin` directory (usually `~/go/bin`). Ensure this directory is in your system's `$PATH`.

Once the binary is installed, you must install the local Chromium browser that `rod-cli` relies on:

```bash
rod-cli install
```

## Option 2: Build from Source

You can also clone the repository and build it manually:

```bash
git clone https://github.com/agenthands/rod-cli.git
cd rod-cli
go build -o rod-cli
sudo mv rod-cli /usr/local/bin/
```

## Verifying Installation

Run the following command to verify the installation:

```bash
rod-cli --version
```
