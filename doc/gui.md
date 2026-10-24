# L'interface

```powershell
.\pm.exe gui
```

Un double-clic sur `pm.exe` fait la même chose : `pm` ouvre son interface dans le navigateur par défaut.
C'est une page servie par `pm` lui-même sur la machine (`127.0.0.1`), sans rien à installer.

Une icône **pm — interface** apparaît dans la zone de notification : « Ouvrir l'interface » rouvre la page, « Quitter l'interface » l'arrête.
Quand plus aucune page de l'interface n'est ouverte depuis 30 secondes, elle s'arrête d'elle-même, sauf si un assistant est en cours : son jeu tourne dans l'interface, et ses réglages d'origine doivent être remis.
Les sessions en cours, elles, continuent toujours quand l'interface s'arrête.
Un second double-clic sur `pm.exe` rouvre la page de l'interface déjà lancée, ou en relance une.

En haut à droite :

- le thème clair ou sombre, gardé par le navigateur ;
- les **réglages** : la **langue** de `pm` (français, anglais, ou celle de Windows), qui vaut pour l'interface, les menus et chaque config qui n'a pas de clé `lang:` (voir [la langue](installation.md#language)), et les dossiers où chercher des configs ;
- les raccourcis clavier : `1`, `2`, `3` pour les onglets, `/` pour filtrer les configs, `Entrée` pour répondre à une question de l'assistant, `Échap` pour fermer une fenêtre ;
- un lien vers cette documentation.

## Sessions {#sessions}

Toutes les sessions `pm run` et `pm watch` en cours, d'où qu'elles viennent (un raccourci, un terminal, l'interface) : leur état, leur PID, leur journal en direct.
Les boutons font ce que fait l'[icône de notification](tray.md) : mettre en pause, reprendre, relancer (pour `pm watch`), **arrêter et finir** (le programme est arrêté, puis les actions de fin sont exécutées), **détacher** (`pm` s'arrête, le programme continue, les actions de fin ne sont pas exécutées).

## Configs {#configs}

L'interface liste :

- les configs dont `pm` se souvient : chacune est notée dès que `pm` la lance, la vérifie, crée ses raccourcis ou la crée avec l'assistant ;
- les `.yaml` des dossiers qu'on lui a donnés, et de ceux qu'elle parcourt d'office : le dossier de `pm.exe` et celui au-dessus, `Saved Games\DCS\Profiles` (et `DCS.openbeta`), et chaque dossier de `Documents\pm`, là où l'assistant range les configs (un seul niveau ; un `.yaml` que `pm` ne sait pas lire n'est pas montré).

On peut ajouter une config ou un dossier par son chemin, et retirer une config de la liste : le fichier, lui, n'est jamais touché.

La liste est à gauche ; la config choisie s'affiche à droite.
En haut, **Lancer** : on choisit le profil et le mode (`run` ou `watch`), et un `pm` démarre à part, comme depuis un raccourci.
Le profil choisi vaut aussi pour les onglets en dessous :

| Onglet | Ce qu'il montre |
|---|---|
| Aperçu | ce que ferait la session, comme [`pm check`](run.md#check) |
| Raccourcis | crée les raccourcis ; si la config ne le dit pas encore, demande le mode, les dossiers, l'affichage et une icône par profil (voir [pm shortcut](shortcut.md)) |
| Journaux | les journaux du profil, à lire dans la page |
| Modifier | modifie la config par formulaires |

« Oublier » retire la config de la liste.
Une config que `pm` refuse est marquée invalide, avec le message qui dit pourquoi, et peut être corrigée dans « Modifier ».

## Modifier une config {#edit}

Chaque clé de la [configuration](configuration.md) a son champ, rangé par thème : programme, session et affichage, variables, fichiers et actions, raccourcis, garder en vie, démarrage terminé, journaux, profils.
Les clés que les formulaires ne montrent pas sont gardées telles quelles.

À l'enregistrement, `pm` relit la nouvelle config avec chaque profil : s'il la refuse, rien n'est écrit et l'erreur s'affiche.
Sinon, l'ancienne version est gardée en `.bak` à côté.
Les commentaires du fichier sont gardés ; la mise en page peut changer (guillemets, alignements).

## Nouvelle config {#wizard}

L'[assistant](new.md) de `pm new`, dans la page : les mêmes questions, les mêmes étapes.
Les réponses déjà données restent affichées à côté ; « Précédent » revient sur la dernière.
Pour un jeu à profils, le jeu démarre quand on clique sur « Lancer le jeu », et son journal défile pendant la partie.
Dans le rapport de chaque profil, « Voir ce qui a changé » montre, pour un fichier texte, les lignes retirées et ajoutées par rapport à l'original ; pour un fichier binaire ou volumineux, sa taille avant et après.

« Abandonner » remet les réglages d'origine et n'écrit rien.

## Sécurité {#security}

La page n'est servie que sur `127.0.0.1`, et seulement au navigateur ouvert par `pm` : l'adresse ouverte contient un jeton secret, que la page garde ensuite dans un cookie qu'aucun autre site ne reçoit.
Un autre programme de la machine, ou un site web ouvert dans le navigateur, ne peut ni lire l'interface ni piloter les sessions.
Chaque session écoute de même sur `127.0.0.1`, avec son propre jeton, écrit dans `%LOCALAPPDATA%\pm\locks`.
