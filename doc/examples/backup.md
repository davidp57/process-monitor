# Exemple : sauvegarder avant de lancer

**Le besoin** : avant chaque partie, garder une copie des sauvegardes d'un jeu, au cas où une partie les abîmerait, sans accumuler les archives à l'infini.

**Ce que fait Process Monitor** :

1. avant le lancement, il crée une archive `.zip` datée du dossier de sauvegardes ;
2. il lance le jeu et attend sa fermeture ;
3. après, il ne garde que les 10 archives les plus récentes.

Si l'archive ne peut pas être créée, le jeu n'est pas lancé : on ne joue pas sans filet.

## La config {#config}

```yaml
--8<-- "examples/backup.yaml"
```

- Les actions `exec` lancent ici PowerShell. Dans une chaîne entre apostrophes YAML, une apostrophe se double (`''`) ; `${archives}` est remplacé par Process Monitor avant l'exécution.
- `timeout: 5m` arrête une commande qui ne rend pas la main.
- Il n'y a pas de `track` : le jeu se lance directement, sans lanceur.

## Lancer {#run}

```powershell
.\pm.exe check .\backup.yaml
.\pm.exe run .\backup.yaml
```

`pm check` montre les deux commandes telles qu'elles seront exécutées, variables remplacées.

!!! note "Un nom de programme unique"
    Process Monitor refuse de démarrer quand un programme du même nom tourne déjà. Pour un jeu, c'est le garde-fou voulu ; mais un programme au nom très courant (`cmd.exe`, `powershell.exe`) serait presque toujours « déjà lancé ». Voir [Dépannage](../troubleshooting.md#already-running).
