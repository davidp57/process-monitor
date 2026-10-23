# Process Monitor

**Process Monitor** lance un programme sous Windows et l'encadre : il prépare ce dont le programme a besoin avant de démarrer, le surveille pendant qu'il tourne, et range derrière lui quand il s'arrête.

C'est un seul fichier, `pm.exe`, sans installation ni dépendance. On le lance d'un raccourci ou d'un terminal, avec un fichier de configuration qui décrit le programme et ce qu'il faut faire autour.

## À quoi il sert {#use-cases}

- **Lancer un jeu avec un profil de réglages, et les récupérer à la fermeture.**
  Plusieurs jeux de réglages pour un même jeu (écran, réalité virtuelle, développement…) : Process Monitor met en place les fichiers du profil choisi, lance le jeu, attend qu'il soit vraiment fermé, puis recopie les réglages modifiés en jeu dans le profil.
  → [Exemple : DCS World par profils](examples/dcs.md)
- **Garder un logiciel en vie.**
  Un utilitaire qui plante de temps en temps, un serveur, un logiciel compagnon : Process Monitor le relance à chaque arrêt, en espaçant les relances s'il plante en boucle.
  → [Exemple : SimAppPro](examples/simapppro.md)
- **Faire le ménage avant et après.**
  Supprimer des fichiers temporaires, vider un cache, lancer une commande ou un script avant le démarrage et après l'arrêt.
  → [Exemple : sauvegarder avant de lancer](examples/backup.md)
- **Fermer ce que le programme a ouvert à côté.**
  Un client radio, un overlay lancés par le jeu : Process Monitor peut les fermer quand le jeu s'arrête.
- **Ne jamais mélanger deux profils.**
  Si le jeu tourne déjà, ou si un autre profil du même jeu est en cours, Process Monitor refuse de démarrer plutôt que d'écraser des réglages.
- **Tourner discrètement.**
  Au lieu d'une fenêtre console, une icône dans la zone de notification, à l'image de votre choix, montre l'état de la session et permet de l'arrêter.

## Ce qu'il fait, en bref {#features}

| Fonction | Ce qu'elle apporte |
|---|---|
| [Profils et fichiers synchronisés](features.md#sync) | une ligne par fichier : copié au lancement, récupéré à la fermeture |
| [Actions avant et après](features.md#actions) | copier, supprimer, lancer une commande |
| [Suivi du vrai programme](features.md#track) | attendre le jeu, pas son lanceur |
| [Garde-fous](features.md#guards) | refus si le programme tourne déjà, verrou entre profils |
| [Relances](features.md#watch) | garder un programme en vie (`pm watch`) |
| [Programmes annexes](features.md#kill-on-exit) | fermer ce que le programme a ouvert |
| [Icône de notification](features.md#tray) | pas de fenêtre console, état en couleur |
| [Journaux et langue](features.md#logs) | un journal par session, en français ou en anglais |

Le détail de chaque fonction et sa configuration sont sur la page [Fonctions](features.md) ; toutes les clés, sur la page [Configuration](configuration.md).

## Démarrage rapide {#quick-start}

1. [Installer](installation.md) `pm.exe`.
2. Écrire une config, en partant d'un [exemple](examples/dcs.md).
3. Vérifier, puis lancer, dans PowerShell :

    ```powershell
    .\pm.exe check .\dcs.yaml --profile DEV
    .\pm.exe run .\dcs.yaml --profile DEV
    ```

Les trois commandes : `pm run` lance une fois avec ses actions avant et après, `pm watch` garde le programme en vie, `pm check` montre ce qui serait fait sans rien lancer.
