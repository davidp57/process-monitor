# pm

`pm` est un exécutable Windows unique, sans dépendance, qui lance un programme et l'encadre.

- **`pm run`** exécute des actions avant le lancement d'un programme, puis d'autres après sa fermeture : copier les fichiers d'un profil, nettoyer, lancer une commande.
- **`pm watch`** garde un programme en vie (watchdog) et le relance s'il s'arrête.
- **`pm check`** valide une config et montre ce qui sera fait, sans rien lancer.

Les deux premières commandes s'affichent dans une console, une interface en terminal ou une simple icône dans la zone de notification.

## Démarrage rapide {#quick-start}

1. [Installer pm](installation.md).
2. Écrire une config. Par exemple, pour lancer un jeu avec les réglages d'un profil et les récupérer à la fermeture :

    ```yaml
    vars:
      saved: ${env:USERPROFILE}\Saved Games\DCS
      prof: ${saved}\Profiles\${profile}
    command: C:\jeux\DCS World\bin\dcs_updater.exe
    track: dcs.exe
    hooks:
      sync:
        - ${prof}\options.lua <-> ${saved}\Config\options.lua
    profiles:
      DEV: {}
      VR: {}
    ```

3. Vérifier, puis lancer, dans PowerShell :

    ```powershell
    .\pm.exe check .\dcs.yaml --profile DEV
    .\pm.exe run .\dcs.yaml --profile DEV
    ```

## Pour aller plus loin {#next}

- [`pm run`](run.md) : le déroulé d'une session et ses garde-fous.
- [`pm watch`](watch.md) : relances et interface en terminal.
- [L'icône de notification](tray.md) : sans fenêtre console.
- [Comment pm suit un programme](tracking.md) : lanceurs, Job Object, ce que pm attend.
- [Configuration](configuration.md) : toutes les clés.
- [Migration depuis la 1.x](migration.md) et [Dépannage](troubleshooting.md).
