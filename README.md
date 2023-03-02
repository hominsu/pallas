<div id="top"></div>

<!-- PROJECT SHIELDS -->
<p align="center">
<a href="https://github.com/hominsu/pallas/graphs/contributors"><img src="https://img.shields.io/github/contributors/hominsu/pallas.svg?style=for-the-badge" alt="Contributors"></a>
<a href="https://github.com/hominsu/pallas/network/members"><img src="https://img.shields.io/github/forks/hominsu/pallas.svg?style=for-the-badge" alt="Forks"></a>
<a href="https://github.com/hominsu/pallas/stargazers"><img src="https://img.shields.io/github/stars/hominsu/pallas.svg?style=for-the-badge" alt="Stargazers"></a>
<a href="https://github.com/hominsu/pallas/issues"><img src="https://img.shields.io/github/issues/hominsu/pallas.svg?style=for-the-badge" alt="Issues"></a>
<a href="https://github.com/hominsu/pallas/blob/master/LICENSE"><img src="https://img.shields.io/github/license/hominsu/pallas.svg?style=for-the-badge" alt="License"></a>
<a href="https://github.com/hominsu/pallas/actions/workflows/docker-publish.yml"><img src="https://img.shields.io/github/actions/workflow/status/hominsu/pallas/go.yml?branch=main&style=for-the-badge" alt="Deploy"></a>
</p>

<!-- PROJECT LOGO -->
<br/>
<div align="center">
<a href="https://github.com/hominsu/pallas">
    <img src="docs/images/pallas.png" alt="Logo" width="196">
</a>

<h3 align="center">pallas</h3>

  <p align="center">
    <br/>
    <a href="https://github.com/hominsu/pallas"><strong>Explore the docs »</strong></a>
    <br/>
    <br/>
    <a href="https://github.com/hominsu/pallas">View Demo</a>
    ·
    <a href="https://github.com/hominsu/pallas/issues">Report Bug</a>
    ·
    <a href="https://github.com/hominsu/pallas/issues">Request Feature</a>
  </p>
</div>

## Build & Contributing

Before building, you need to have `GO >= 1.18`, [Buf CLI](https://docs.buf.build/installation). If you are developing on windows, use [scoop](https://github.com/ScoopInstaller/Scoop) to install `busybox` and `make`

#### Clone this repository

```bash
git clone https://github.com/hominsu/pallas.git
```

#### Initial Workspace

```bash
go work init && go work use -r ./app && go mod tidy
```

#### Install dependencies

```bash
make init
```

#### Generate other code

```bash
make api && make conf && make ent && make wire
```

#### Compile

```bash
make build
```