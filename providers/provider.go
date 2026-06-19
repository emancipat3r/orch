package providers

import (
	"context"
	"strconv"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/utils"
)

// Option is a selectable choice in the create wizard. Label is what the user
// sees; Value is the underlying id/slug handed to the provider API.
type Option struct {
	Label string
	Value string
}

// CreateSpec carries everything needed to provision an instance, decoupling the
// create command from each provider's bespoke function signature.
type CreateSpec struct {
	Image        string // image/OS id or slug (Option.Value from Images)
	Region       string // region id or slug (Option.Value from Regions)
	Size         string // type/plan id or slug (Option.Value from Sizes)
	SSHKeyID     string
	PrivKeyPath  string
	VPSName      string
	InstanceFile string
}

// Instance is a provider-agnostic view of a running VPS, used for list/destroy
// so callers never have to know provider-specific response shapes.
type Instance struct {
	ID      string
	IP      string
	Image   string
	Region  string
	Type    string
	Created string
	Status  string
}

// Provider abstracts a cloud VPS provider. Each concrete provider wraps its
// existing API client functions so the cmd layer can stay provider-agnostic.
type Provider interface {
	Name() string
	Balance(ctx context.Context) (string, error)
	Regions(ctx context.Context) ([]Option, error)
	Images(ctx context.Context) ([]Option, error)
	Sizes(ctx context.Context) ([]Option, error)
	UploadSSHKey(ctx context.Context, pubKeyPath string) (string, error)
	Create(ctx context.Context, spec CreateSpec) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	Destroy(ctx context.Context, instanceID, instanceFile string) error
}

// GetProvider returns a Provider for the given display name (as produced by
// ui.ChoiceProvider), loading and validating its API key from configFile.
func GetProvider(name, configFile string) (Provider, error) {
	key := utils.ParseCreds(configFile, name)
	if key == "" {
		return nil, logger.Errorf("missing or invalid API key for %s", name)
	}

	switch name {
	case "Linode":
		return &linodeProvider{key: key}, nil
	case "DigitalOcean":
		return &doProvider{key: key}, nil
	case "Vultr":
		return &vultrProvider{key: key}, nil
	default:
		return nil, logger.Errorf("unknown provider: %s", name)
	}
}

// ---------------- Linode ----------------

type linodeProvider struct{ key string }

func (p *linodeProvider) Name() string { return "Linode" }

func (p *linodeProvider) Balance(ctx context.Context) (string, error) {
	return GetLinodesBalance(p.key)
}

func (p *linodeProvider) Regions(ctx context.Context) ([]Option, error) {
	regions, err := GetLinodeRegions()
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, r := range regions {
		if r.Status != "ok" {
			continue
		}
		opts = append(opts, Option{Label: r.ID + " - " + r.Label, Value: r.ID})
	}
	return opts, nil
}

func (p *linodeProvider) Images(ctx context.Context) ([]Option, error) {
	images, err := GetLinodeImages()
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, i := range images {
		opts = append(opts, Option{Label: i.ID + " - " + i.Label, Value: i.ID})
	}
	return opts, nil
}

func (p *linodeProvider) Sizes(ctx context.Context) ([]Option, error) {
	resources, err := GetLinodeResources()
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, r := range resources {
		opts = append(opts, Option{Label: r.ID + " - " + r.Label, Value: r.ID})
	}
	return opts, nil
}

func (p *linodeProvider) UploadSSHKey(ctx context.Context, pubKeyPath string) (string, error) {
	id, err := UploadLinodeSSHKey(p.key, pubKeyPath)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(id), nil
}

func (p *linodeProvider) Create(ctx context.Context, spec CreateSpec) (Instance, error) {
	keyID, err := strconv.Atoi(spec.SSHKeyID)
	if err != nil {
		return Instance{}, logger.Errorf("invalid ssh key id %q: %v", spec.SSHKeyID, err)
	}
	rootPass, err := utils.GenerateRandomPassword(30)
	if err != nil {
		return Instance{}, logger.Errorf("generate root password: %v", err)
	}
	id, err := CreateLinode(ctx, p.key, keyID, spec.PrivKeyPath, spec.Image, spec.Region, spec.Size, rootPass, spec.InstanceFile, spec.VPSName)
	if err != nil {
		return Instance{}, err
	}
	return Instance{ID: id}, nil
}

func (p *linodeProvider) List(ctx context.Context) ([]Instance, error) {
	raw, err := SelectLinodeInstance(p.key)
	if err != nil {
		return nil, err
	}
	regionCache := map[string]string{}
	if regions, err := GetLinodeRegions(); err == nil {
		for _, r := range regions {
			regionCache[r.ID] = r.ID + " - " + r.Label
		}
	}
	var out []Instance
	for _, r := range raw {
		ip := ""
		if len(r.Ipv4) > 0 {
			ip = r.Ipv4[0]
		}
		region := r.Region
		if v, ok := regionCache[r.Region]; ok {
			region = v
		}
		out = append(out, Instance{
			ID:      strconv.Itoa(r.Id),
			IP:      ip,
			Image:   r.Host_Image,
			Region:  region,
			Type:    r.Type,
			Created: r.Creation_Time,
			Status:  r.Status,
		})
	}
	return out, nil
}

func (p *linodeProvider) Destroy(ctx context.Context, instanceID, instanceFile string) error {
	_, err := DestroyLinode(p.key, instanceID, instanceFile)
	return err
}

// ---------------- DigitalOcean ----------------

type doProvider struct{ key string }

func (p *doProvider) Name() string { return "DigitalOcean" }

func (p *doProvider) Balance(ctx context.Context) (string, error) {
	return GetDOBalance(p.key)
}

func (p *doProvider) Regions(ctx context.Context) ([]Option, error) {
	regions, err := GetDORegions(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, r := range regions {
		opts = append(opts, Option{Label: r.Slug + " - " + r.Name, Value: r.Slug})
	}
	return opts, nil
}

func (p *doProvider) Images(ctx context.Context) ([]Option, error) {
	images, err := GetDOImages(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, i := range images {
		opts = append(opts, Option{Label: i.Slug + " - " + i.Name, Value: i.Slug})
	}
	return opts, nil
}

func (p *doProvider) Sizes(ctx context.Context) ([]Option, error) {
	sizes, err := GetDOResources(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, s := range sizes {
		opts = append(opts, Option{Label: s.Slug + " - " + s.Name, Value: s.Slug})
	}
	return opts, nil
}

func (p *doProvider) UploadSSHKey(ctx context.Context, pubKeyPath string) (string, error) {
	id, err := UploadDOSSHKey(p.key, pubKeyPath)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(id), nil
}

func (p *doProvider) Create(ctx context.Context, spec CreateSpec) (Instance, error) {
	keyID, err := strconv.Atoi(spec.SSHKeyID)
	if err != nil {
		return Instance{}, logger.Errorf("invalid ssh key id %q: %v", spec.SSHKeyID, err)
	}
	id, err := CreateDroplet(ctx, p.key, keyID, spec.PrivKeyPath, spec.Image, spec.Region, spec.Size, spec.InstanceFile, spec.VPSName)
	if err != nil {
		return Instance{}, err
	}
	return Instance{ID: id}, nil
}

func (p *doProvider) List(ctx context.Context) ([]Instance, error) {
	raw, err := SelectDOInstance(p.key)
	if err != nil {
		return nil, err
	}
	var out []Instance
	for _, d := range raw {
		ip := ""
		for _, v4 := range d.Networks.IPv4 {
			if v4.Type == "public" {
				ip = v4.IPAddress
				break
			}
		}
		image := d.Image.Description
		if image == "" {
			image = d.Image.Name
		}
		if image == "" {
			image = d.Image.Slug
		}
		out = append(out, Instance{
			ID:      strconv.Itoa(d.Id),
			IP:      ip,
			Image:   image,
			Region:  d.Region.Slug + " - " + d.Region.Name,
			Type:    d.Size.Slug,
			Created: d.Creation_Time,
			Status:  d.Status,
		})
	}
	return out, nil
}

func (p *doProvider) Destroy(ctx context.Context, instanceID, instanceFile string) error {
	_, err := DestroyDroplet(p.key, instanceID, instanceFile)
	return err
}

// ---------------- Vultr ----------------

type vultrProvider struct{ key string }

func (p *vultrProvider) Name() string { return "Vultr" }

func (p *vultrProvider) Balance(ctx context.Context) (string, error) {
	return GetVultrBalance(p.key)
}

func (p *vultrProvider) Regions(ctx context.Context) ([]Option, error) {
	regions, err := GetVultrRegions(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, r := range regions {
		opts = append(opts, Option{Label: r.ID + " - " + r.City + ", " + r.Country, Value: r.ID})
	}
	return opts, nil
}

func (p *vultrProvider) Images(ctx context.Context) ([]Option, error) {
	osImages, err := GetVultrOS(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, o := range osImages {
		opts = append(opts, Option{Label: strconv.Itoa(o.ID) + " - " + o.Name, Value: strconv.Itoa(o.ID)})
	}
	return opts, nil
}

func (p *vultrProvider) Sizes(ctx context.Context) ([]Option, error) {
	plans, err := GetVultrPlans(p.key)
	if err != nil {
		return nil, err
	}
	var opts []Option
	for _, pl := range plans {
		label := pl.ID + " - " + strconv.Itoa(pl.VCPUCount) + " vCPU, " +
			strconv.Itoa(pl.RAM) + " MB RAM, " + strconv.Itoa(pl.Disk) + " GB SSD - $" +
			strconv.FormatFloat(pl.MonthlyCost, 'f', 2, 64) + "/month"
		opts = append(opts, Option{Label: label, Value: pl.ID})
	}
	return opts, nil
}

func (p *vultrProvider) UploadSSHKey(ctx context.Context, pubKeyPath string) (string, error) {
	return UploadVultrSSHKey(p.key, pubKeyPath)
}

func (p *vultrProvider) Create(ctx context.Context, spec CreateSpec) (Instance, error) {
	osID, err := strconv.Atoi(spec.Image)
	if err != nil {
		return Instance{}, logger.Errorf("invalid os id %q: %v", spec.Image, err)
	}
	id, err := CreateVultrInstance(ctx, p.key, spec.SSHKeyID, spec.PrivKeyPath, osID, spec.Region, spec.Size, spec.InstanceFile, spec.VPSName)
	if err != nil {
		return Instance{}, err
	}
	return Instance{ID: id}, nil
}

func (p *vultrProvider) List(ctx context.Context) ([]Instance, error) {
	raw, err := ListVultrInstances(p.key)
	if err != nil {
		return nil, err
	}
	regionCache := map[string]string{}
	if regions, err := GetVultrRegions(p.key); err == nil {
		for _, r := range regions {
			regionCache[r.ID] = r.ID + " - " + r.City + ", " + r.Country
		}
	}
	var out []Instance
	for _, v := range raw {
		region := v.Region
		if r, ok := regionCache[v.Region]; ok {
			region = r
		}
		out = append(out, Instance{
			ID:      v.ID,
			IP:      v.MainIP,
			Image:   v.OS,
			Region:  region,
			Type:    v.Plan,
			Created: v.DateCreated,
			Status:  v.Status,
		})
	}
	return out, nil
}

func (p *vultrProvider) Destroy(ctx context.Context, instanceID, instanceFile string) error {
	return DestroyVultr(p.key, instanceID, instanceFile)
}
