# L'icône de notification

`--tray`, ou `ui: tray` dans la config, remplace la fenêtre par une icône dans la zone de notification, pour `pm run` comme pour `pm watch`.

Lancé depuis un raccourci, pm rend sa fenêtre console dès le démarrage (réglez le raccourci sur « Réduite » pour qu'elle ne s'affiche même pas).
Lancé depuis un terminal, il continue d'y écrire, et le double Ctrl+C reste actif.

## Ce que montre l'icône {#display}

- **Couleur** : verte quand le programme tourne, orange pendant les copies (au début et à la fin) ou avant une relance, grise en pause, rouge après une erreur.
- **Infobulle** : la session, le profil et l'état.
- **Image** : `--icon FICHIER`, ou `icon:` dans la config, ou par profil dans `profiles:`, remplace le disque par votre image (`.png`, ou `.ico` en 32 bits ou contenant un PNG), avec une petite pastille de la couleur de l'état en bas à droite. Une image illisible est signalée dans le journal, et pm garde le disque.

## Le menu {#menu}

- **Ouvrir le journal**.
- **Arrêter et finir** : arrête le programme, puis exécute les actions *après*.
- **Détacher…** : après une confirmation qui dit ce que ça coûte à ce moment-là (annuler au démarrage, terminer pendant le délai de relance, sinon laisser le programme sans copie retour). Si la session change d'état pendant que la question est affichée, elle est reposée.
- En `watch` : **Mettre en pause**, **Reprendre**, **Relancer**.

## La fin de la session {#end}

- En cas d'échec (erreur, ou code de sortie non nul en `pm run`), l'icône passe au rouge et une fenêtre Windows dit ce qui a échoué et où est le journal.
- Une annulation demandée n'est pas un échec : pas de fenêtre d'erreur.
- Sinon l'icône garde son état final pendant `exit.delay`, puis disparaît ; avec `exit.wait_key: always`, une fenêtre « Session terminée » s'affiche.

Les commandes `exec` ne s'ouvrent pas dans une fenêtre console, ni le programme lancé quand pm n'a plus de console.
