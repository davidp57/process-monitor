# Comment pm suit un programme

Le programme est lancé dans un **Job Object** Windows : tous les process qu'il crée y restent rattachés.
Sans `track`, la session se termine quand le dernier process du groupe s'est arrêté.

## Les lanceurs {#launchers}

Beaucoup de programmes passent par un lanceur : `dcs_updater.exe` démarre `dcs.exe` puis se ferme.
`track: dcs.exe` dit à pm de suivre ce process-là, par son handle, et non le lanceur :

- pm le trouve dans le groupe dès sa création ;
- s'il a quitté le groupe, pm le cherche par son nom parmi les process créés après le lancement ;
- quand il s'arrête, pm attend encore 3 secondes, au cas où il passerait la main à un autre process du même nom (le Notepad de Windows 11 fait ainsi) ;
- si le **lanceur** tourne encore (pour relancer le jeu après une mise à jour, par exemple), pm l'attend et le signale dans le journal ;
- les autres process que le jeu a lancés (un client radio comme SRS, un overlay) ne sont **pas** attendus : le journal les nomme, et `kill_on_exit` peut les arrêter ;
- s'il n'apparaît pas dans les 10 secondes qui suivent la fin de tout le reste, la session se termine en erreur.

## Garde-fous {#guards}

- **Verrous** : une session prend un verrou à son `name`, et un par fichier live de ses lignes `sync`. Deux sessions de même nom, ou deux configs qui écrivent le même fichier live, ne peuvent pas tourner ensemble : la seconde est refusée avant toute copie. Windows libère les verrous à la sortie de pm, même après un plantage.
- **Programme déjà lancé hors de pm** (`if_running`), vérifié sur le process suivi **et** sur le lanceur : `abort` (défaut) refuse, rien n'est copié ni lancé ; `kill` arrête les instances existantes, le lanceur d'abord, puis vérifie que plus rien ne tourne.
- **Dernier filet** (avec `track`) : si, au moment des actions de fin, le process suivi tourne encore sans être suivi, ou si la liste des process est illisible, les actions de fin ne sont pas exécutées et la session finit en erreur. Mieux vaut une copie retour manquée, signalée, qu'un profil écrasé pendant que le jeu tourne.
- **Copies sûres** : écrites dans un fichier temporaire puis renommées, jamais à moitié écrites ; l'ancien fichier est gardé en `.bak`.

## Journaux {#logs}

Chaque session écrit `<name>[-<profil>]-<date>-<heure>.log` dans `log.dir` : étapes de pm, copies, sortie du programme si `capture`.
Seuls les `log.keep` plus récents sont gardés.
