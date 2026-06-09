{
  description = "Bitbucket Pipeline TUI — infra-pipeline-ui";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        # Package: build the Go binary
        packages.default = pkgs.buildGoModule {
          pname = "infra-pipeline-ui";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-hipaq95BFUCZhax+kkLGok99+ZMnaODJ2bDdiAVO+9A=";
          ldflags = [ "-s" "-w" ];
          meta = with pkgs.lib; {
            description = "Bitbucket Pipeline TUI";
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
          ];

          shellHook = ''
            echo "🛠  infra-pipeline-ui dev shell"
            echo "   go build  → build binary"
            echo "   go run .  → run TUI"
            echo "   go test ./...  → test"
            echo "   golangci-lint run  → lint"
            echo "   nix build  → build via nix"
            echo "   nix run    → run via nix"
          '';
        };

        # Convenience scripts
        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/infra-pipeline-ui";
        };
      }
    );
}