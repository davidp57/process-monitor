# Exemple : garder SimAppPro en vie

**Le besoin** : SimAppPro, le logiciel qui pilote des périphériques de simulation, doit tourner pendant toute la session de jeu ; s'il plante, il faut le relancer, et ses fichiers temporaires s'accumulent.

**Ce que fait Process Monitor** :

1. il arrête une instance de SimAppPro déjà lancée, pour repartir d'un état connu ;
2. il nettoie les fichiers temporaires, puis lance SimAppPro ;
3. à chaque arrêt, il le relance, en nettoyant d'abord ; s'il plante en boucle, il espace les relances jusqu'à une minute ;
4. quand on arrête la session, il nettoie une dernière fois.

## La config {#config}

```yaml
--8<-- "examples/simapppro.yaml"
```

- `if_running: kill` : une instance déjà lancée est arrêtée au lieu de bloquer le démarrage.
- `capture: true` et `ELECTRON_NO_ATTACH_CONSOLE` : la sortie de SimAppPro va dans le journal, sans ouvrir de console.
- `watch` : relance toujours, 2 secondes après l'arrêt, jusqu'à une minute en cas de plantages en série.
- `hooks.before` est rejoué avant chaque relance ; `hooks.after` ne s'exécute qu'à la fin de la session.
- `ui: tray` et `icon:` : l'icône de SimAppPro dans la zone de notification, avec la pastille d'état. Si le fichier d'image n'existe pas chez vous, pm le dit dans le journal et garde le disque de couleur.

## Lancer {#run}

```powershell
.\pm.exe watch .\simapppro.yaml
```

Le menu de l'icône permet de mettre en pause, de relancer, d'arrêter ou de détacher. En console, sans `ui: tray`, les mêmes actions sont sur les touches `p`, `r`, `k` et `q` (voir [pm watch](../watch.md)).
