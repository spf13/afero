{ pkgs, ... }:

{
  packages = with pkgs; [
    golangci-lint
  ];

  languages = {
    go = {
      enable = true;

      package = pkgs.go_1_27;
    };
  };
}
