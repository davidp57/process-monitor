# Dépannage

## PowerShell et les chemins {#powershell}

Les exemples de cette doc sont écrits pour **PowerShell**, le terminal par défaut de Windows.

- **Lancer pm** : écrire `.\pm.exe`, pas `pm.exe`. PowerShell ne cherche pas les programmes dans le dossier courant ; `cmd.exe` accepte les deux formes, `.\` est donc la seule qui marche partout.
- **Variables d'environnement** : `$env:USERPROFILE` en PowerShell, `%USERPROFILE%` dans `cmd.exe`. Écrite dans PowerShell, `%USERPROFILE%` arrive telle quelle à pm, qui cherche alors un dossier nommé `%USERPROFILE%` :

    ```powershell
    .\pm.exe run "$env:USERPROFILE\Saved Games\DCS\Profiles\pm-dcs.yaml" --profile DEV
    ```

- Dans la **config** YAML, c'est pm qui lit les variables, avec sa propre syntaxe : `${env:USERPROFILE}`.

## « une autre session pm est déjà active » {#busy}

Une session de même `name`, ou qui écrit les mêmes fichiers live, tourne déjà ; le message donne son profil.
Le verrou disparaît avec le process pm qui le tient, même après un plantage : s'il persiste, un pm tourne encore (voir l'icône de notification, ou le Gestionnaire des tâches).

## « … tourne déjà hors de cette session » {#already-running}

Le jeu, ou son lanceur, a été lancé sans pm. pm refuse pour ne pas écraser les réglages en cours : fermer le programme, ou mettre `if_running: kill` si c'est voulu.

La vérification se fait par le nom du programme : un nom très courant (`cmd.exe`, `powershell.exe`) est presque toujours « déjà lancé ». Lancer directement le vrai programme : `track` ne contourne pas ce refus, car le programme lancé est vérifié lui aussi.

## Les copies retour n'ont pas eu lieu {#no-copy-back}

Le journal dit pourquoi : session détachée (double Ctrl+C, `q` deux fois, fenêtre fermée), ou dernier filet (le process suivi tournait encore sans être suivi).
Les fichiers live, eux, n'ont pas bougé : relancer la session puis la terminer normalement recopie les réglages.

## Un process lancé par le jeu reste ouvert {#leftovers}

pm n'attend que le jeu et son lanceur ; les autres process (un client radio, un overlay) sont nommés dans le journal.
Pour les arrêter à la fin, les lister dans `kill_on_exit`. Un programme qui tourne en administrateur ne peut pas être arrêté par un pm lancé sans élévation : le journal affiche « accès refusé ».

## L'icône n'affiche pas mon image {#icon}

pm lit les `.png` et les `.ico` en 32 bits ou contenant un PNG. Un `.ico` en 256 couleurs est refusé : le journal le dit, et pm garde le disque de couleur. Utiliser le `.png` correspondant, ou réenregistrer l'icône en 32 bits.

## pm parle la mauvaise langue {#language}

pm suit la langue d'affichage de Windows. Pour la forcer : `--lang fr`, la variable `PM_LANG`, ou `lang: fr` dans la config.
