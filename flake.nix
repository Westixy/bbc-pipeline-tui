{
  description = "Bitbucket Pipeline TUI — infra-pipeline-ui (Go + Svelte webapp)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Build the Svelte webapp frontend (produces dist/ files)
        webapp = pkgs.buildNpmPackage {
          pname = "bbc-pipeline-webapp";
          version = "1.0.0";
          src = ./webapp;
          npmDepsHash = "sha256-iycuonvJ1TQMAYBrOXK24nkCWY8hLLKapVXjkaOwoBc=";
          dontNpmBuild = true;
          buildPhase = ''
            runHook preBuild
            npm run build
            runHook postBuild
          '';
          installPhase = ''
            runHook preInstall
            mkdir -p $out
            cp -r "$NIX_BUILD_TOP/server/webapp-dist/"* "$out/"
            runHook postInstall
          '';
        };
      in
      {
        # Package: build the Go binary (with embedded webapp frontend)
        packages.default = pkgs.buildGoModule {
          pname = "infra-pipeline-ui";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-hipaq95BFUCZhax+kkLGok99+ZMnaODJ2bDdiAVO+9A=";
          ldflags = [ "-s" "-w" ];

          preBuild = ''
            # Copy pre-built webapp distribution into the path expected by go:embed
            mkdir -p server/webapp-dist
            cp -r ${webapp}/* server/webapp-dist/
          '';

          meta = with pkgs.lib; {
            description = "Bitbucket Pipeline TUI with embedded web UI";
            homepage = "https://github.com/bbc/infra-pipeline-ui";
            license = licenses.mit;
            mainProgram = "infra-pipeline-ui";
          };
        };

        # Development shell
        devShells.default = pkgs.mkShell {
          name = "infra-pipeline-ui-dev";

          buildInputs = with pkgs; [
            go
            golangci-lint
            gnumake
            nodejs
          ];

          shellHook = ''
            echo "🛠  infra-pipeline-ui dev shell"
            echo "   go build  → build binary (webapp must be pre-built)"
            echo "   go run .  → run TUI"
            echo "   go test ./...  → test"
            echo "   golangci-lint run  → lint"
            echo "   nix build  → build via nix (Go + frontend)"
            echo "   nix run    → run via nix"
            echo ""
            echo "   (cd webapp && npm install && npm run dev)  → start webapp dev server"
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/infra-pipeline-ui";
        };
      }
    );
}
