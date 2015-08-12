package mikro

import (
  "log"
  "net/url"
  "time"
  "strconv"

  "github.com/AdRoll/goamz/aws"
  "github.com/AdRoll/goamz/dynamodb"

  "github.com/gliderlabs/registrator/bridge"
)

var tableDescription = dynamodb.TableDescriptionT{
  AttributeDefinitions: []dynamodb.AttributeDefinitionT{
    dynamodb.AttributeDefinitionT{"Service", "S"},
    dynamodb.AttributeDefinitionT{"TaskID", "S"},
    },
  KeySchema: []dynamodb.KeySchemaT{
    dynamodb.KeySchemaT{"Service", "HASH"},
    dynamodb.KeySchemaT{"TaskID", "RANGE"},
    },
}

func init() {
  bridge.Register(new(Factory), "mikro")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
  pk, err := tableDescription.BuildPrimaryKey()
  if err != nil {
    log.Fatal(err);
  }

  auth, err := aws.GetAuth("", "", "", time.Now())

  dbServer := dynamodb.New(auth, aws.GetRegion("eu-west-1"))
  table := dbServer.NewTable(uri.Host, pk)
  return &MikroAdapter{table: table}
}

type MikroAdapter struct {
  table *dynamodb.Table
}

func (r *MikroAdapter) Ping() error {
  return nil
}

func (r *MikroAdapter) Register(service *bridge.Service) error {
  attrs := []dynamodb.Attribute{
    //*dynamodb.NewStringAttribute("InstanceID", instanceId),
    *dynamodb.NewStringAttribute("HostAddr", service.IP),
    *dynamodb.NewNumericAttribute("HostPort", strconv.Itoa(service.Port)),
    *dynamodb.NewStringAttribute("ExpiresAt", time.Now().Add(
      time.Second * time.Duration(service.TTL)).Format(time.RFC3339)),
    *dynamodb.NewStringAttribute("UpdatedAt", time.Now().Format(time.RFC3339)),
  }

  ok, err := r.table.PutItem(service.Name, service.ID, attrs)
  if !ok {
    log.Println("mikro: failed to register service:", err)
  }
  return err
}

func (r *MikroAdapter) Deregister(service *bridge.Service) error {
  ok, err := r.table.DeleteItem(&dynamodb.Key{service.Name, service.ID})
  if !ok {
    log.Println("mikro: failed to deregister service:", err)
  }
  return err
}

func (r *MikroAdapter) Refresh(service *bridge.Service) error {
  return r.Register(service)
}
