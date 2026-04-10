{
  description = "nebu-sql - SQL over installed nebu processor binaries";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.packageOverrides = prev: {
            duckdb = prev.duckdb.overrideAttrs (_: rec {
              version = "1.5.1";
              src = prev.fetchFromGitHub {
                owner = "duckdb";
                repo = "duckdb";
                rev = "v${version}";
                hash = "sha256-FygBpfhvezvUbI969Dta+vZOPt6BnSW2d5gO4I4oB2A=";
              };
            });
          };
        };
        versionString = if self ? shortRev then self.shortRev else "dev";
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_25
            gopls
            gotools
            go-tools

            # Release tooling
            goreleaser

            # Build tools
            gnumake

            # Query / smoke-test tools
            duckdb
            jq
            git
          ];

          shellHook = ''
            export PS1='\n\[\033[1;35m\][nebu-sql]\[\033[0m\] \[\033[1;32m\]\w\[\033[0m\] \$ '

            echo "🚀 nebu-sql development environment"
            echo ""
            echo "Available tools:"
            echo "  go version: $(go version)"
            echo "  duckdb version: $(duckdb --version)"
            echo "  goreleaser version: $(goreleaser --version | head -1)"
            echo ""
            echo "Quick start:"
            echo "  make test        - Run tests"
            echo "  make build       - Build the binary"
            echo "  make smoke-real  - Smoke-test real processors"
            echo ""
            echo "Note: real-processor smoke tests expect 'nebu' on PATH so processors can be installed via 'nebu install'."
            echo ""
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "nebu-sql";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-Nwy0guUvdW6Q5qsxHTX5wmOWnDaPfOKJZJNVJ334b0k=";

          subPackages = [
            "cmd/nebu-sql"
          ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/withObsrvr/nebu-sql/internal/version.Value=${versionString}"
          ];

          meta = with pkgs.lib; {
            description = "SQL over installed nebu processor binaries";
            homepage = "https://github.com/withObsrvr/nebu-sql";
            license = licenses.asl20;
            mainProgram = "nebu-sql";
          };
        };
      });
}
