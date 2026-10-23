# pm run

```powershell
.\pm.exe run <config.yaml> [--profile NOM] [--tray] [--icon FICHIER] [--lang fr|en]
```

## Déroulé {#flow}

1. **Verrou de session** : une seule session par `name`, et une seule par fichier live (voir [garde-fous](tracking.md#guards)).
2. **Programme déjà lancé ?** pm refuse, ou l'arrête avec `if_running: kill`.
3. **Actions avant** : les lignes `sync` copient les fichiers du profil vers le live, puis les actions `before` s'exécutent.
4. **Lancement**, puis **attente de la fin** du programme suivi (voir [comment pm suit un programme](tracking.md)).
5. **Actions après** : les lignes `sync` recopient le live vers le profil, puis les actions `after` s'exécutent.

Si une action *avant* échoue, rien n'est lancé, et les fichiers live déjà remplacés sont remis comme ils étaient.
Si une action *après* échoue, les suivantes sont quand même exécutées, et l'erreur est affichée.

Chaque commande `exec` est limitée à `hooks.timeout` (10 minutes par défaut) et s'exécute dans le dossier de la config ; un programme qu'elle laisse en arrière-plan ne bloque pas pm.

## Ctrl+C et la fenêtre {#interrupt}

Ctrl+C ne coupe pas tout de suite : il faut un second Ctrl+C dans les 5 secondes, et la console dit d'abord ce qu'il fera :

- pendant les copies de départ, il **annule** : rien n'est lancé, les fichiers live sont remis comme avant ;
- pendant que le programme tourne, il **détache** pm : le programme continue sans lui, et les actions *après* ne sont pas exécutées ;
- en `watch`, pendant le délai avant une relance, il **termine** la session, actions *après* comprises.

Fermer la fenêtre de la console coupe pm sans confirmation possible : les actions *après* n'ont pas lieu.
Pour ne pas avoir de fenêtre à fermer par mégarde, utiliser l'[icône de notification](tray.md).

## Codes de sortie {#exit-codes}

| Code | Sens |
|---|---|
| celui du programme | la session s'est déroulée ; pm rend le code du programme suivi |
| 1 | pm a rencontré une erreur, ou s'est détaché (la copie retour n'a pas eu lieu) |
| 2 | erreur de config ou de ligne de commande |

## Avant de lancer : pm check {#check}

`.\pm.exe check <config.yaml> [--profile NOM]` ne lance rien : il affiche ce qui sera copié, supprimé ou exécuté, et signale les fichiers source absents.
À passer après chaque modification de config.
